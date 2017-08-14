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

    function executeSql(sql) {
        $.ajax({
            type: 'POST',
            url: pathname + "/query",
            data: {tid: activeMerchantId, sql: sql},
            success: function (content, textStatus, request) {
                tableCreate(content, sql)
            },
            error: function (jqXHR, textStatus, errorThrown) {
                alert(jqXHR.responseText + "\nStatus: " + textStatus + "\nError: " + errorThrown);
            }
        })
        hideTablesDiv()
    }

    $('.executeQuery').prop("disabled", true).click(function () {
        var sql = codeMirror.somethingSelected() ? codeMirror.getSelection() : codeMirror.getValue()
        executeSql(sql)
    })

    var queryResultId = 0;

    function createDelete(result) {
        return 'delete from ' + result.TableName + ' '
    }

    var regex = new RegExp(/[\0\x08\x09\x1a\n\r"'\\\%]/g)
    var escaper = function escaper(char) {
        var m = ['\\0', '\\x08', '\\x09', '\\x1a', '\\n', '\\r', "'", '"', "\\", '\\\\', "%"];
        var r = ['\\\\0', '\\\\b', '\\\\t', '\\\\z', '\\\\n', '\\\\r', "''", '""', '\\\\', '\\\\\\\\', '\\%'];
        return r[m.indexOf(char)];
    }

    function createInsert(cells, result) {
        var insertSql = 'insert into ' + wrapFieldName(result.TableName) + '('
        for (var i = 0; i < result.Headers.length; ++i) {
            if (i > 0) {
                insertSql += ', '
            }
            insertSql += wrapFieldName(result.Headers[i])
        }
        insertSql += ') values ('

        cells.each(function (jndex, cell) {
            if (jndex > 1) {
                insertSql += ', '
            }
            if (jndex > 0) {
                var newValue = $(cell).text()
                if ("(null)" == newValue) {
                    insertSql += 'null'
                } else {
                    insertSql += '\'' + newValue.replace(regex, escaper) + '\''
                }
            }
        })
        return insertSql + ')'
    }

    function createUpdateSetPart(cells, result, headRow) {
        var updateSql = null
        cells.each(function (jndex, cell) {
            var oldValue = $(this).attr('old')
            if (oldValue) {
                if (updateSql == null) {
                    updateSql = 'update ' + wrapFieldName(result.TableName) + ' set '
                } else {
                    updateSql += ', '
                }
                var fieldName = $(headRow.get(jndex + 1)).text()
                var newValue = $(cell).text()
                if ("(null)" == newValue) {
                    updateSql += wrapFieldName(fieldName) + ' is null'
                } else {
                    updateSql += wrapFieldName(fieldName) + ' = \'' + newValue.replace(regex, escaper) + '\''
                }
            }
        })
        return updateSql
    }

    function wrapFieldName(fieldName) {
        if (fieldName.indexOf('_') >= 0) return fieldName
        else return '`' + fieldName + '`'
    }

    function createWherePart(updateSql, result, headRow, cells) {
        updateSql += ' where '
        if (result.PrimaryKeysIndex.length > 0) {
            for (var i = 0; i < result.PrimaryKeysIndex.length; ++i) {
                var ki = result.PrimaryKeysIndex[i] + 1
                if (i > 0) {
                    updateSql += ' and '
                }
                var pkName = $(headRow.get(ki + 1)).text()
                var $cell = $(cells.get(ki));
                var pkValue = $cell.attr('old') || $cell.text()
                updateSql += wrapFieldName(pkName) + ' = \'' + pkValue.replace(regex, escaper) + '\''
            }
            return updateSql
        } else {
            var wherePart = ''
            cells.each(function (jndex, cell) {
                if (jndex > 0) {
                    var whereValue = $(this).attr('old') || $(cell).text()
                    if (wherePart != '') {
                        wherePart += ' and '
                    }
                    var fieldName = $(headRow.get(jndex + 1)).text()

                    if ("(null)" == whereValue) {
                        wherePart += wrapFieldName(fieldName) + ' is null'
                    } else {
                        wherePart += wrapFieldName(fieldName) + ' = \'' + whereValue.replace(regex, escaper) + '\''
                    }
                }
            })
            if (wherePart != null) {
                updateSql += wherePart
            }
        }

        return updateSql
    }

    function executeUpdate(sqlRowIndices, sqls, $rows) {
        $.ajax({
            type: 'POST',
            url: pathname + "/update",
            data: {tid: activeMerchantId, sqls: sqls},
            success: function (content, textStatus, request) {
                if (!content.Ok) {
                    alert(content.Message)
                    return
                }

                for (var i = 0; i < content.RowsResult.length; ++i) {
                    var rowResult = content.RowsResult[i]
                    if (!rowResult.Ok) {
                        alert(rowResult.Message)
                    } else {
                        var rowIndex = sqlRowIndices[i]
                        var $row = $($rows[rowIndex])

                        $row.find('td.dataCell').each(function (jndex, cell) {
                            $(this).removeAttr('old').removeClass('changedCell')
                        })
                        $row.find('input[type=checkbox]').prop('checked', false)
                        $row.remove('.deletedRow').removeClass('clonedRow')
                    }
                }
            },
            error: function (jqXHR, textStatus, errorThrown) {
                alert(jqXHR.responseText + "\nStatus: " + textStatus + "\nError: " + errorThrown);
            }
        })
    }

    function attachSaveUpdatesEvent(result) {
        $('#saveUpdates' + queryResultId).click(function (event) {
            var table = $(this).parents('div').next('table')
            var headRow = table.find('tr.headRow').first().find('td')

            var sqls = []
            var sqlRowIndices = []
            var $rows = table.find('tr.dataRow');
            $rows.each(function (index, row) {
                var $row = $(row);
                var cells = $row.find('td.dataCell')
                if ($row.hasClass('clonedRow')) {
                    var insertSql = createInsert(cells, result)
                    sqls[sqls.length] = insertSql
                    sqlRowIndices[sqlRowIndices.length] = index
                } else if ($row.hasClass('deletedRow')) {
                    var deleteSql = 'delete from ' + result.TableName + ' '
                    deleteSql = createWherePart(deleteSql, result, headRow, cells)
                    sqls[sqls.length] = deleteSql
                    sqlRowIndices[sqlRowIndices.length] = index
                } else {
                    var updateSql = createUpdateSetPart(cells, result, headRow)

                    if (updateSql != null) {
                        updateSql = createWherePart(updateSql, result, headRow, cells)
                        sqls[sqls.length] = updateSql
                        sqlRowIndices[sqlRowIndices.length] = index
                    }
                }
            })
            if (sqls.length == 0) {
                alert('There is no changes to save!')
                return
            }

            var joinedSqls = sqls.join(';\n')
            if (confirm(joinedSqls + ';\n\nAre you sure to save ?')) {
                executeUpdate(sqlRowIndices, joinedSqls, $rows)
            }
        })
    }

    function alternateRowsColor() {
        $('#queryResult' + queryResultId + ' tr:even').addClass('rowEven')
    }

    function toggleRowEditable(event) {
        var rowChecked = $(this).prop('checked')
        var dataCells = $(this).parents('tr').find('td.dataCell')
        if (!rowChecked) {
            dataCells.attr('contenteditable', false)
                .unbind('dblclick').unbind('blur')
            return
        }

        dataCells.dblclick(function (event) {
            var $this = $(this)
            if (!$this.attr('old')) {
                $this.attr('old', $this.text())
            }
            $this.attr('contenteditable', true)
                .focus()
                .keydown(function (event) {
                    var keyCode = event.keyCode || event.which
                    if (keyCode == 13 && event.ctrlKey) {
                        $this.blur()
                    }
                })
        }).blur(function (event) {
            var $this = $(this)
            $this.attr('contenteditable', false)
            if ($this.attr('old') == $this.text()) {
                $this.removeAttr('old').removeClass('changedCell')
            } else {
                $this.addClass('changedCell')
            }

            $this.toggleClass('nullCell', '(null)' == $this.text())
        })
    }

    function checkboxEditableChange(checkboxEditable) {
        var edittable = checkboxEditable.prop('checked')
        checkboxEditable.parent().find('span.editButtons').toggle(edittable)
        var dataTable = checkboxEditable.parent().next('table');
        dataTable.find('.chk').toggle(edittable)
        var rowCheckboxes = dataTable.find('.dataRow').find('input[type=checkbox]')
        rowCheckboxes.unbind('click')
        if (edittable) {
            rowCheckboxes.click(toggleRowEditable)
        }
    }

    function attachEditableEvent() {
        var checkboxEditable = $('#checkboxEditable' + queryResultId)
        checkboxEditableChange(checkboxEditable)
        checkboxEditable.click(function () {
            checkboxEditableChange(checkboxEditable)
        })
    }

    function copyRow($tr) {
        $tr.find(':checked').prop("checked", false)
        var $clone = $tr.clone().addClass('clonedRow');
        $clone.insertAfter($tr)
        $clone.find('input[type=checkbox]').click(toggleRowEditable).click()
    }

    function attachDeleteRowsEvent() {
        $('#deleteRows' + queryResultId).click(function () {
            var checkboxes = $('#queryResult' + queryResultId + ' :checked')
            checkboxes.parents('tr').addClass('deletedRow')
        })
    }

    function attachCopyRowEvent() {
        $('#copyRow' + queryResultId).click(function () {
            var checkboxes = $('#queryResult' + queryResultId + ' :checked')
            if (checkboxes.length == 0) {
                alert('please specify which row to copy')
            } else if (checkboxes.length > 1) {
                alert('please specify only one row to copy')
            } else {
                copyRow($(checkboxes[0]).parents('tr'))
            }
        })
    }

    function tableCreate(result, sql) {
        var rowUpdateReady = result.TableName && result.TableName != ""

        ++queryResultId
        var table = '<table class="executionSummary"><tr><td>time</td><td>cost</td><td>sql</td><td>error</td></tr>'
            + '<tr><td>' + result.ExecutionTime + '</td><td>' + result.CostTime + '</td><td>' + sql + '</td><td'
            + (result.Error && (' class="error">' + result.Error) || '>OK')
            + '</td><tr></table><br/>'

        if (rowUpdateReady) {
            table += '<div><input type="checkbox" id="checkboxEditable' + queryResultId + '" class="checkboxEditable">'
                + '<label for="checkboxEditable' + queryResultId + '">Editable?</label>'
                + '<span class="editButtons"><button id="copyRow' + queryResultId + '" class="copyRow">Copy Row</button>'
                + '<button id="deleteRows' + queryResultId + '">Delete Rows</button>'
                + '<button id="saveUpdates' + queryResultId + '">Save To DB</button>'
                + '</span></div>'
        }

        table += '<table id="queryResult' + queryResultId + '" class="queryResult">'

        if (result.Headers && result.Headers.length > 0) {
            table += '<tr class="headRow" queryResultId="' + queryResultId + '">'
            if (rowUpdateReady) {
                table += '<td><div class="chk checkAll"></div></td>'
            }
            table += '<td>#</td><td>' + result.Headers.join('</td><td>') + '</td></tr>'
        }
        if (result.Rows && result.Rows.length > 0) {
            for (var i = 0; i < result.Rows.length; i++) {
                table += '<tr class="dataRow">'
                if (rowUpdateReady) {
                    table += '<td><div class="chk checkMe"><input type="checkbox"></div></td>'
                }

                for (var j = 0; j < result.Rows[i].length; ++j) {
                    var cellValue = result.Rows[i][j]
                    if ('(null)' == cellValue) {
                        table += '<td class="dataCell nullCell">' + cellValue + '</td>'
                    } else {
                        table += '<td class="dataCell">' + cellValue + '</td>'
                    }
                }

                table += '</tr>'
            }
        } else if (result.Rows && result.Rows.length == 0) {
            table += '<tr class="dataRow clonedRow">'
            if (rowUpdateReady) {
                table += '<td><div class="chk checkMe"><input type="checkbox"></div></td>'
            }
            table += '<td class="dataCell">' + new Array(result.Headers.length + 1).join('</td><td class="dataCell">') + '</td></tr>'
        }
        table += '</table><br/>'
        $(table).prependTo($('.result'))

        alternateRowsColor()

        if (rowUpdateReady) {
            attachEditableEvent()
            attachCopyRowEvent()
            attachDeleteRowsEvent()
            attachSaveUpdatesEvent(result)
        }
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
            },
            error: function (jqXHR, textStatus, errorThrown) {
                alert(jqXHR.responseText + "\nStatus: " + textStatus + "\nError: " + errorThrown);
            }
        })
    })

    function showTables(result) {
        var resultHtml = ''
        if (result.Rows && result.Rows.length > 0) {
            for (var i = 0; i < result.Rows.length; i++) {
                resultHtml += '<span>' + result.Rows[i][1] + '</span>'
            }
        }
        $('.tables').html(resultHtml)
    }

    function showTablesAjax(activeMerchantId) {
        $.ajax({
            type: 'POST',
            url: pathname + "/query",
            data: {tid: activeMerchantId, sql: 'show tables'},
            success: function (content, textStatus, request) {
                showTables(content)
                showTablesDiv()
            },
            error: function (jqXHR, textStatus, errorThrown) {
                alert(jqXHR.responseText + "\nStatus: " + textStatus + "\nError: " + errorThrown);
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
            hideTablesDiv()
        } else {
            $button.data('alreadyclicked', true)
            var alreadyclickedTimeout = setTimeout(function () {
                $button.data('alreadyclicked', false) // reset when it happens
                executeSql('select * from ' + tableName)
                hideTablesDiv()
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

    $('.loginButton').click(function () {
        window.location = pathname + "/login"
    })

    function hideTablesDiv() {
        $('.tablesWrapper').hide()
        $('.hideTables').text('Show Tables')
    }

    function showTablesDiv() {
        $('.tablesWrapper').show()
        $('.hideTables').text('Hide Tables')
    }
})
()