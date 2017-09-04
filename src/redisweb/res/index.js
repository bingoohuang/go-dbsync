$(function () {
    var pathname = window.location.pathname
    if (pathname.lastIndexOf("/", pathname.length - 1) !== -1) {
        pathname = pathname.substring(0, pathname.length - 1)
    }

    function refreshKeys(key) {
        $.ajax({
            type: 'GET',
            url: pathname + "/listKeys",
            data: {server: $('#servers').val(), database: $('#databases').val(), pattern: $('#serverFilterKeys').val()},
            success: function (content, textStatus, request) {
                showKeysTree(content)
                if (key) {
                    chosenKey(key)
                }
            },
            error: function (jqXHR, textStatus, errorThrown) {
                alert(jqXHR.responseText + "\nStatus: " + textStatus + "\nError: " + errorThrown)
            }
        })
    }

    refreshKeys();

    $('#serverFilterKeysBtn,#refreshKeys').click(function () {
        refreshKeys()
    })

    function executeRedisCmd() {
        var cmd = $('#directCmd').val()
        var server = $('#servers').val()
        $.ajax({
            type: 'POST',
            url: pathname + "/redisCli",
            data: {server: server, database: $('#databases').val(), cmd: cmd},
            success: function (result, textStatus, request) {
                var resultHtml = '<pre>' + server + '&gt; ' + cmd + '</pre>' +
                    '<pre>' + result + '</pre>'

                $('#directCmdResult').prepend(resultHtml)
            },
            error: function (jqXHR, textStatus, errorThrown) {
                alert(jqXHR.responseText + "\nStatus: " + textStatus + "\nError: " + errorThrown)
            }
        })
    }

    $('#redisTerminal').click(function () {
        var contentHtml = '<div><input id="directCmd" placeholder="input commands"><button id="redisTerminalBtn">Execute</button></div>' +
            '<div id="directCmdResult"></div>'
        $('#frame').html(contentHtml)

        $('#directCmd').keydown(function (event) {
            var keyCode = event.keyCode || event.which
            if (keyCode == 13) {
                executeRedisCmd()
            }
        })
        $('#redisTerminalBtn').click(executeRedisCmd)


    })

    $('#redisInfo').click(function () {
        $.ajax({
            type: 'GET',
            url: pathname + "/redisInfo",
            data: {server: $('#servers').val(), database: $('#databases').val()},
            success: function (result, textStatus, request) {
                var contentHtml = '<div><span class="key">Redis info</span></div>' +
                    '<pre>' + result + '</pre>'

                $('#frame').html(contentHtml)
            },
            error: function (jqXHR, textStatus, errorThrown) {
                alert(jqXHR.responseText + "\nStatus: " + textStatus + "\nError: " + errorThrown)
            }
        })
    })

    function showKeysTree(keysArray) {
        $('#keysNum').html('(' + keysArray.length + ')')
        var keysHtml = '<ul>'
        for (var i = 0; i < keysArray.length; ++i) {
            var key = keysArray[i]
            var nodeCss = i < keysArray.length - 1 ? "sprite-tree-node" : "sprite-tree-lastnode last"
            keysHtml += '<li class="datatype-' + key.Type + ' sprite ' + nodeCss + '" data-type="' + key.Type + '">' +
                '<span class="sprite sprite-datatype-' + key.Type + '"></span><span class="keyValue">' + key.Key + '</span></li>'
        }
        keysHtml += '</ul>'

        $('#keys').html(keysHtml)

        $('#keys ul li').click(function () {
            $('#keys ul li').removeClass('chosen')
            var $this = $(this)
            $this.addClass('chosen')
            var key = $this.find('.keyValue').text()
            var type = $this.attr('data-type')
            $.ajax({
                type: 'GET',
                url: pathname + "/showContent",
                data: {server: $('#servers').val(), database: $('#databases').val(), key: key, type: type},
                success: function (result, textStatus, request) {
                    showContent(key, type, result.Content, result.Ttl, result.Size, result.Encoding, result.Error, result.Exists, result.Format)
                },
                error: function (jqXHR, textStatus, errorThrown) {
                    alert(jqXHR.responseText + "\nStatus: " + textStatus + "\nError: " + errorThrown)
                }
            })
        })
        toggleFilterKeys()
    }

    function toggleFilterKeys() {
        var filter = $.trim($('#filterKeys').val()).toUpperCase()

        $('#keys ul li').each(function (index, li) {
            var $li = $(li)
            var text = $.trim($li.text()).toUpperCase()
            var contains = text.indexOf(filter) > -1
            $li.toggle(contains)
        })
    }

    $('#filterKeys').keyup(toggleFilterKeys)

    function chosenKey(key) {
        $('#keys ul li').removeClass('chosen').each(function (index, li) {
            var $span = $(li).find('.keyValue')
            if ($span.text() == key) {
                $(li).addClass('chosen')
                return false
            }
        })
    }

    function removeKey(key) {
        $('#keys ul li').removeClass('chosen').each(function (index, li) {
            var $span = $(li).find('.keyValue')
            if ($span.text() == key) {
                $(li).remove()
                return false
            }
        })
    }


    $('#servers,#databases').change(refreshKeys)

    function showContentAjax(key, type) {
        $.ajax({
            type: 'GET',
            url: pathname + "/showContent",
            data: {server: $('#servers').val(), database: $('#databases').val(), key: key, type: type},
            success: function (result, textStatus, request) {
                showContent(key, type, result.Content, result.Ttl, result.Size, result.Encoding, result.Error, result.Exists, result.Format)
            },
            error: function (jqXHR, textStatus, errorThrown) {
                alert(jqXHR.responseText + "\nStatus: " + textStatus + "\nError: " + errorThrown)
            }
        })
    }

    $('#addKey').click(function () {
        var contentHtml = '<div><span class="key">Add another key</span></div>'
        contentHtml += '<table class="contentTable">' +
            '<tr><td class="titleCell">Type:</td><td colspan="2"><select name="type" id="type">' +
            '<option value="string">String</option><option value="hash">Hash</option><option value="list">List</option><option value="set">Set</option><option value="zset">Sorted Set</option>' +
            '</select></td></tr>' +
            '<tr><td class="titleCell">Key:</td><td colspan="2"><input id="key" placeholder="input the new key"></td></tr>' +
            '<tr><td class="titleCell">TTL:</td><td colspan="2"><input id="ttl" placeholder="input the expired time, like 1d/1h/10s/-1s"></td></tr>' +
            '<tr><td class="titleCell">Format:</td><td colspan="2"><select name="format" id="format">' +
            '<option value="String">String</option><option value="JSON">JSON</option><option value="Quoted">Quoted</option>' +
            '</select></td></tr>' +
            '<tr><td class="titleCell">Value:</td><td colspan="2"><span class="valueSave sprite sprite-save"></span></td></tr>'

        contentHtml += '<tr class="newKeyTr string"><td colspan="2"><textarea id="code"></textarea></td></tr>'

        contentHtml += '<tr class="newKeyTr hash"><td class="titleCell">Field</td><td colspan="2" class="titleCell">Value</td></tr>'
        for (var i = 0; i < 10; ++i) {
            contentHtml += '<tr class="newKeyTr hash"><td contenteditable="true"></td><td colspan="2" contenteditable="true"></td></tr>'
        }

        contentHtml += '<tr class="newKeyTr list set"><td class="titleCell">#</td><td colspan="2" class="titleCell">Value</td></tr>'
        for (var i = 0; i < 10; ++i) {
            contentHtml += '<tr class="newKeyTr list set"><td>' + i + '</td><td colspan="2" contenteditable="true"></td></tr>'
        }

        contentHtml += '<tr class="newKeyTr zset"><td class="titleCell">#</td><td class="titleCell">Score</td><td class="titleCell">Value</td></tr>'
        for (var i = 0; i < 10; ++i) {
            contentHtml += '<tr class="newKeyTr zset"><td>' + i + '</td><td contenteditable="true"></td><td contenteditable="true"></td></tr>'
        }

        contentHtml += '</table>'
        contentHtml += '<button id="addMoreRowsBtn">Add More Rows</button>'

        $('#frame').html(contentHtml)

        $('tr.newKeyTr').hide()
        $('tr.string').show()
        $('#addMoreRowsBtn').hide().click(function () {
            var type = $('#type').val()
            var rows = $('tr.' + type)
            var startRowNum = rows.length - 1

            var moreRows = ''
            for (var i = 0; i < 10; ++i) {
                if (type == 'hash') {
                    moreRows += '<tr class="newKeyTr hash"><td contenteditable="true"></td><td colspan="2" contenteditable="true"></td></tr>'
                } else if (type == 'list' || type == 'set') {
                    moreRows += '<tr class="newKeyTr list set"><td>' + (startRowNum + i) + '</td><td colspan="2" contenteditable="true"></td></tr>'
                } else if (type == 'zset') {
                    moreRows += '<tr class="newKeyTr zset"><td>' + (startRowNum + i) + '</td><td contenteditable="true"></td><td contenteditable="true"></td></tr>'
                }
            }
            $(moreRows).appendTo($('.contentTable'))
        })


        $('#type').change(function () {
            var type = $('#type').val()
            $('tr.newKeyTr').hide()
            $('tr.' + type).show()
            $('#addMoreRowsBtn').toggle(type != 'string')
        })

        $('#format').change(function () {
            codeMirror = null
            if ($(this).val() == 'JSON' && $('#type').val() == 'string') {
                codeMirror = CodeMirror.fromTextArea(document.getElementById('code'), {
                    mode: 'application/json',
                    lineNumbers: true,
                    matchBrackets: true
                })
            }
        })

        $('.valueSave').click(function () {
            var type = $('#type').val()
            var key = $('#key').val()
            var ttl = $('#ttl').val()
            var format = $('#format').val()
            var value = "" 
            if (type == 'string') {
                value = codeMirror != null && codeMirror.getValue() || $('#code').val()
            } else if (type == 'hash') {
                $('tr.hash').gt(0).each(function () {
                    
                })
            }

            if (confirm("Are you sure to save save for " + key + "?")) {
                $.ajax({
                    type: 'POST',
                    url: pathname + "/newKey",
                    data: {
                        server: $('#servers').val(),
                        database: $('#databases').val(),
                        type: type,
                        key: key,
                        ttl: ttl,
                        format: format,
                        value: value
                    },
                    success: function (content, textStatus, request) {
                        if (content == 'OK') {
                            refreshKeys(key)
                            showContentAjax(key, type)
                        } else {
                            alert(content)
                        }
                    },
                    error: function (jqXHR, textStatus, errorThrown) {
                        alert(jqXHR.responseText + "\nStatus: " + textStatus + "\nError: " + errorThrown)
                    }
                })
            }
        })
    })

    var codeMirror = null

    function showContent(key, type, content, ttl, size, encoding, error, exists, format) {
        if (error != "") {
            contentHtml = '<div><span class="error">' + error + '</span></div>'
            $('#frame').html(contentHtml)
        }

        if (!exists) {
            contentHtml = '<div><span class="key">' + key + ' does not exits</span></div>'
            $('#frame').html(contentHtml)
            return
        }

        var contentHtml = '<div><span class="key">' + key + '</span></div>'
        contentHtml += '<table class="contentTable">' +
            '<tr><td class="titleCell">Type:</td><td colspan="2">' + type + '</td></tr>' +
            '<tr><td class="titleCell">TTL:</td><td colspan="2">' + ttl + '</td></tr>' +
            '<tr><td class="titleCell">Encoding:</td><td colspan="2">' + encoding + '</td></tr>' +
            '<tr><td class="titleCell">Format:</td><td colspan="2">' + format + '</td></tr>' +
            '<tr><td class="titleCell">Size:</td><td colspan="2">' + size + '</td></tr>' +
            '<tr><td class="titleCell">Value:</td><td colspan="2"><span class="valueSave sprite sprite-save"></span><span class="keyDelete sprite sprite-delete"></span></td></tr>'

        switch (type) {
            case "string":
                contentHtml += '<tr><td colspan="3"><textarea id="code">' + content + '</textarea></td></tr>'
                break
            case "hash":
                contentHtml += '<tr><td class="titleCell">Field</td><td class="titleCell" colspan="2">Value</td></tr>'
                for (var key in content) {
                    contentHtml += '<tr><td contenteditable="true">' + key + '</td><td colspan="2" contenteditable="true">' + content[key] + '</td></tr>'
                }
                break
            case "set":
            case "list":
                contentHtml += '<tr><td class="titleCell">#</td><td class="titleCell" colspan="2">Value</td></tr>'
                for (var i = 0; i < content.length; ++i) {
                    contentHtml += '<tr><td contenteditable="true">' + i + '</td><td colspan="2" contenteditable="true">' + content[i] + '</td></tr>'
                }
                break
            case "zset":
                contentHtml += '<tr><td class="titleCell">#</td><td class="titleCell">Score</td><td class="titleCell">Value</td></tr>'
                for (var i = 0; i < content.length; ++i) {
                    contentHtml += '<tr><td contenteditable="true">' + i + '</td><td contenteditable="true">' + content[i].Score + '</td><td>' + content[i].Member + '</td></tr>'
                }
                break

        }
        contentHtml += '</table>'

        $('#frame').html(contentHtml)

        codeMirror = null
        if (format === "JSON") {
            codeMirror = CodeMirror.fromTextArea(document.getElementById('code'), {
                mode: 'application/json',
                lineNumbers: true,
                matchBrackets: true
            })
        } else {
            autosize($('#code'));
        }

        $('.keyDelete').click(function () {
            if (confirm("Are you sure to delete " + key + "?")) {
                $.ajax({
                    type: 'POST',
                    url: pathname + "/deleteKey",
                    data: {server: $('#servers').val(), database: $('#databases').val(), key: key},
                    success: function (content, textStatus, request) {
                        if (content != 'OK') {
                            alert(content)
                            return
                        }

                        removeKey(key)

                        contentHtml = '<div><span class="key">' + key + ' does not exits</span></div>'
                        $('#frame').html(contentHtml)
                    },
                    error: function (jqXHR, textStatus, errorThrown) {
                        alert(jqXHR.responseText + "\nStatus: " + textStatus + "\nError: " + errorThrown)
                    }
                })
            }
        })

        $('.valueSave').click(function () {
            if (confirm("Are you sure to save changes for " + key + "?")) {
                var changedContent = codeMirror != null && codeMirror.getValue() || $('#code').val()
                $.ajax({
                    type: 'POST',
                    url: pathname + "/changeContent",
                    data: {
                        server: $('#servers').val(),
                        database: $('#databases').val(),
                        key: key,
                        changedContent: changedContent,
                        format: format
                    },
                    success: function (content, textStatus, request) {
                        if (content == 'OK') {
                            showContentAjax(key, type)
                        } else {
                            alert(content)
                        }
                    },
                    error: function (jqXHR, textStatus, errorThrown) {
                        alert(jqXHR.responseText + "\nStatus: " + textStatus + "\nError: " + errorThrown)
                    }
                })
            }
        })
    }

})