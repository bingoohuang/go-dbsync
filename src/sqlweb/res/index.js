(function () {
    var MIN_HEIGHT = 60
    var start_y
    var start_h

    function on_drag(e) {
        var newHeight = Math.max(MIN_HEIGHT, (start_h + e.y - start_y)) + "px"
        codeMirror.setSize(null, newHeight)
    }

    function on_release(e) {
        document.body.removeEventListener("mousemove", on_drag)
        window.removeEventListener("mouseup", on_release)
    }

    $('.resizeHandle')[0].addEventListener("mousedown", function (e) {
        start_y = e.y
        start_h = $('.CodeMirror').height()
        document.body.addEventListener("mousemove", on_drag)
        window.addEventListener("mouseup", on_release)
    })

    var mac = CodeMirror.keyMap.default == CodeMirror.keyMap.macDefault // 判断是否为Mac
    var runKey = (mac ? "Cmd" : "Ctrl") + "-Enter"
    var extraKeys = {}
    extraKeys[runKey] = function (cm) {
        var executeQuery = $('.executeQuery')
        if (!executeQuery.prop("disabled")) executeQuery.click()
    }

    var codeMirror = CodeMirror.fromTextArea(document.getElementById('code'), {
        mode: 'text/x-mysql',
        indentWithTabs: true,
        smartIndent: true,
        lineNumbers: true,
        matchBrackets: true,
        extraKeys: extraKeys
    })
    codeMirror.setSize(null, '60px')

    $('.collapseSql').click(function () {
        codeMirror.setSize(null, '60px')
    })

    var pathname = window.location.pathname
    if (pathname.lastIndexOf("/", pathname.length - 1) !== -1) {
        pathname = pathname.substring(0, pathname.length - 1)
    }

    var executeSql = function (sql) {
        $.ajax({
            type: 'POST',
            url: pathname + "/query",
            data: {tid: activeMerchantId, sql: sql},
            success: function (content, textStatus, request) {
                tableCreate(content, sql)
            }
        })
    }

    $('.executeQuery').prop("disabled", true).click(function () {
        var sql = codeMirror.somethingSelected() ? codeMirror.getSelection() : codeMirror.getValue()
        executeSql(sql)
    })

    var queryResultId = 0;

    function attachQueryResultEditableEvent() {
        $('#queryResult' + queryResultId + ' td.dataCell')
            .dblclick(function (event) {
                if (!$(this).attr('old')) {
                    $(this).attr('old', $(this).text())
                }
                $(this).attr('contenteditable', true).focus()
            })
            .blur(function (event) {
                $(this).attr('contenteditable', false)
                if ($(this).attr('old') == $(this).text()) {
                    $(this).removeAttr('old').removeClass('changedCell')
                } else {
                    $(this).addClass('changedCell')
                }
                toggleUpdatesActivate($(this).parents('table').prev('button.saveUpdates'))
            })
    }

    function createUpdateSetPart(cells, result, headRow) {
        var updateSql = null
        cells.each(function (jndex, cell) {
            var oldValue = $(this).attr('old')
            if (oldValue) {
                var newValue = $(cell).text()
                if (updateSql == null) {
                    updateSql = 'update ' + result.TableName + ' set '
                } else {
                    updateSql += ', '
                }

                var fieldName = $(headRow.get(jndex)).text()
                updateSql += fieldName + ' = "' + newValue + '"'
            }
        })
        return updateSql
    }

    function createUpdateWherePart(updateSql, result, headRow, cells) {
        updateSql += ' where '
        for (var i = 0; i < result.PrimaryKeysIndex.length; ++i) {
            var ki = result.PrimaryKeysIndex[i] + 1
            if (i > 0) {
                updateSql += ', '
            }
            var pkName = $(headRow.get(ki)).text()
            var pkValue = $(cells.get(ki)).attr('old') || $(cells.get(ki)).text()
            updateSql += pkName + ' = "' + pkValue + '"'


        }
        return updateSql
    }

    function executeUpdate(updateSql, cells, $saveUpdatesButton) {
        $.ajax({
            type: 'POST',
            url: pathname + "/query",
            data: {tid: activeMerchantId, sql: updateSql},
            success: function (content, textStatus, request) {
                cells.each(function (jndex, cell) {
                    $(this).removeAttr('old').removeClass('changedCell')
                    toggleUpdatesActivate($saveUpdatesButton)
                })
            }
        })
    }

    function attachSaveUpdatesEvent(result) {
        $('#saveUpdates' + queryResultId).click(function (event) {
            var $this = $(this);
            var headRow = $this.next('table').find('tr.headRow').first().find('td')
            var rows = $this.next('table').find('tr.dataRow')

            rows.each(function (index, row) {
                var cells = $(row).find('td.dataCell')
                var updateSql = createUpdateSetPart(cells, result, headRow)

                if (updateSql != null) {
                    updateSql = createUpdateWherePart(updateSql, result, headRow, cells)
                }

                if (updateSql != null) {
                    executeUpdate(updateSql, cells, $this)
                }
            })
        })
    }

    function attackSaveUpdatesClick(result) {
        attachQueryResultEditableEvent()
        attachSaveUpdatesEvent(result)
    }

    var alternateRowsColor = function () {
        $('#queryResult' + queryResultId + ' tr:even').addClass('rowEven')
    }

    function tableCreate(result, sql) {
        var rowUpdateReady = result.RowUpdateReady;

        ++queryResultId
        var table = '<table class="executionSummary"><tr><td>time</td><td>cost</td><td>sql</td><td>error</td></tr>'
            + '<tr><td>' + result.ExecutionTime + '</td><td>' + result.CostTime + '</td><td>' + sql + '</td><td'
            + (result.Error && (' class="error">' + result.Error) || '>OK')
            + '</td><tr></table><br/>'
            + '<button id="saveUpdates' + queryResultId + '" class="saveUpdates">save updates</button>'
            + '<table id="queryResult' + queryResultId + '" class="queryResult">'

        if (result.Headers && result.Headers.length > 0) {
            table += '<tr class="headRow"><td>#</td><td>' + result.Headers.join('</td><td>') + '</td></tr>'
        }
        if (result.Rows && result.Rows.length > 0) {
            for (var i = 0; i < result.Rows.length; i++) {
                table += '<tr class="dataRow"><td class="dataCell">' + result.Rows[i].join('</td><td class="dataCell">') +'</td></tr>'
            }
        } else if (result.Rows && result.Rows.length == 0) {
            table += '<tr><td>-</td><td colspan="' + result.Headers.length + '">0 rows returned</td></tr>'
        }
        table += '</table><br/>'
        $(table).prependTo($('.result'))

        alternateRowsColor()

        if (rowUpdateReady === true) {
            attackSaveUpdatesClick(result)
        }
    }

    var toggleUpdatesActivate = function ($saveUpdatesButton) {
        var show = false
        $saveUpdatesButton.next('table').find('tr.dataRow')
            .each(function (index, row) {
                var cells = $(row).find('td.dataCell')
                cells.each(function (jndex, cell) {
                    if ($(this).attr('old')) {
                        show = true
                        return false
                    }
                })
            })
        $saveUpdatesButton.toggle(show)
    }

    $('.clearResult').click(function () {
        $('.result').html('')
    })

    $('.searchKey').keydown(function (event) {
        var keyCode = event.keyCode || event.which
        if (keyCode == 13) $('.searchButton').click()
    })

    $('.searchButton').click(function () {
        hideTablesDiv()
        $.ajax({
            type: 'POST',
            url: pathname + "/searchDb",
            data: {searchKey: $('.searchKey').val()},
            success: function (content, textStatus, request) {
                var searchResult = $('.searchResult')
                var searchHtml = ''
                if (content && content.length) {
                    for (var j = 0; j < content.length; j++) {
                        searchHtml += '<span tid="' + content[j].MerchantId + '">🌀' + content[j].MerchantName + '</span>'
                    }
                } else {
                    $('.executeQuery').prop("disabled", true)
                    $('.tables').html('')
                }
                searchResult.html(searchHtml)
                $('.searchResult span:first-child').click()
            }
        })
    })

    var showTables = function (result) {
        var resultHtml = ''
        if (result.Rows && result.Rows.length > 0) {
            for (var i = 0; i < result.Rows.length; i++) {
                resultHtml += '<span>' + result.Rows[i][1] + '</span>'
            }
        }
        $('.tables').html(resultHtml)
    }

    var showTablesAjax = function (activeMerchantId) {
        $.ajax({
            type: 'POST',
            url: pathname + "/query",
            data: {tid: activeMerchantId, sql: 'show tables'},
            success: function (content, textStatus, request) {
                showTables(content)
                showTablesDiv()
            }
        })
    }

    $('.tables').on('click', 'span', function (event) {
        var $button = $(this)
        var tableName = $(this).text()
        if ($button.data('alreadyclicked')) {
            $button.data('alreadyclicked', false) // reset
            if ($button.data('alreadyclickedTimeout')) {
                clearTimeout($button.data('alreadyclickedTimeout')) // prevent this from happening
            }
            executeSql('show full columns from ' + tableName)
            $('.hideTables').click()
        } else {
            $button.data('alreadyclicked', true)
            var alreadyclickedTimeout = setTimeout(function () {
                $button.data('alreadyclicked', false) // reset when it happens
                executeSql('select * from ' + tableName)
                $('.hideTables').click()
            }, 300) // <-- dblclick tolerance here
            $button.data('alreadyclickedTimeout', alreadyclickedTimeout) // store this id to clear if necessary
        }
        return false
    })

    var activeMerchantId = null
    $('.searchResult').on('click', 'span', function () {
        $('.searchResult span').removeClass('active')
        $(this).addClass('active')
        activeMerchantId = $(this).attr('tid')
        $('.executeQuery').prop("disabled", false)
        showTablesAjax(activeMerchantId)
    })

    $('.formatSql').click(function () {
        var sql = codeMirror.somethingSelected() ? codeMirror.getSelection() : codeMirror.getValue()
        var formattedSql = sqlFormatter.format(sql, {language: 'sql'})
        codeMirror.setValue(formattedSql)
    })
    $('.clearSql').click(function () {
        codeMirror.setValue('')
    })
    $('.hideTables').click(function () {
        var visible = $('.tablesWrapper').toggle($(this).text() != 'Hide Tables').is(":visible")
        $(this).text(visible ? 'Hide Tables' : 'Show Tables')
    })
    var hideTablesDiv = function () {
        $('.tablesWrapper').hide()
        $('.hideTables').text('Show Tables')
    }
    var showTablesDiv = function () {
        $('.tablesWrapper').show()
        $('.hideTables').text('Hide Tables')
    }
})()