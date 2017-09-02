$(function () {
    var pathname = window.location.pathname
    if (pathname.lastIndexOf("/", pathname.length - 1) !== -1) {
        pathname = pathname.substring(0, pathname.length - 1)
    }

    function refreshKeys() {
        $.ajax({
            type: 'GET',
            url: pathname + "/listKeys",
            data: {server: $('#servers').val(), database: $('#databases').val()},
            success: function (content, textStatus, request) {
                showKeysTree(content)
            },
            error: function (jqXHR, textStatus, errorThrown) {
                alert(jqXHR.responseText + "\nStatus: " + textStatus + "\nError: " + errorThrown)
            }
        })
    }

    refreshKeys();

    function showKeysTree(keysArray) {
        $('#keysNum').html('(' + keysArray.length + ')')
        var keysHtml = '<ul>'
        for (var i = 0; i < keysArray.length; ++i) {
            var key = keysArray[i]
            if (i < keysArray.length - 1) {
                keysHtml += '<li class="' + key.Type + ' sprite sprite-tree-node" data-type="' + key.Type + '">' + key.Key + '</li>'
            } else {
                keysHtml += '<li class="' + key.Type + ' sprite sprite-tree-lastnode last" data-type="' + key.Type + '">' + key.Key + '</li>'
            }
        }
        keysHtml += '</ul>'

        $('#keys').html(keysHtml)

        $('#keys ul li').click(function () {
            var $this = $(this)
            var key = $this.text()
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
    }

    $('.filterKeys').keyup(function () {
        var filter = $.trim($(this).val()).toUpperCase()

        $('#keys ul li').each(function (index, li) {
            var $li = $(li)
            var text = $.trim($li.text()).toUpperCase()
            var contains = text.indexOf(filter) > -1
            $li.toggle(contains)
        })
    })


    $('#servers,#databases').change(refreshKeys)

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
        contentHtml += '<table>' +
            '<tr><td>Type:</td><td>' + type + '</td></tr>' +
            '<tr><td>TTL:</td><td>' + ttl + '</td></tr>' +
            '<tr><td>Encoding:</td><td>' + encoding + '</td></tr>' +
            '<tr><td>Format:</td><td>' + format + '</td></tr>' +
            '<tr><td>Size:</td><td>' + size + '</td></tr>' +
            '<tr><td>Value:</td><td><span class="valueSave sprite sprite-save"></span><span class="keyDelete sprite sprite-delete"></span></td></tr>' +
            '<tr><td colspan="2"><textarea id="code">' + content + '</textarea></td></tr>' +
            '</table>'

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
                    data: {server: $('#servers').val(), database: $('#databases').val(), key: key, changedContent: changedContent, format: format},
                    success: function (content, textStatus, request) {
                        alert(content)
                    },
                    error: function (jqXHR, textStatus, errorThrown) {
                        alert(jqXHR.responseText + "\nStatus: " + textStatus + "\nError: " + errorThrown)
                    }
                })
            }
        })
    }


    var isResizing = false;
    var lastDownX = 0;
    var lastWidth = 0;

    var resizeSidebar = function (w) {
        $('#sidebar').css('width', w);
        $('#keys').css('width', w);
        $('#resize').css('left', w + 10);
        $('#resize-layover').css('left', w + 15);
        $('#frame').css('left', w + 15);
    };

    if (parseInt(Cookies.get('sidebar')) > 0) {
        resizeSidebar(parseInt(Cookies.get('sidebar')));
    }

    $('#resize').on('mousedown', function (e) {
        isResizing = true;
        lastDownX = e.clientX;
        lastWidth = $('#sidebar').width();
        $('#resize-layover').css('z-index', 1000);
        e.preventDefault();
    });
    $(document).on('mousemove', function (e) {
        if (!isResizing) {
            return;
        }

        var w = lastWidth - (lastDownX - e.clientX);
        if (w < 250) {
            w = 250;
        } else if (w > 1000) {
            w = 1000;
        }

        resizeSidebar(w);
        Cookies.set('sidebar', w);
    }).on('mouseup', function (e) {
        isResizing = false;
        $('#resize-layover').css('z-index', 0);
    });
})