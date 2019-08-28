$.get("navbar.page", function (data) {
    $("#navbar").html(data);
    var url = window.location.pathname;
    if (url == "/") {
        url ="/index.html";
    }
    $('ul.nav a[href="' + url + '"]').parent().addClass('active');
    $('ul.nav a').filter(function () {
        return this.href.pathname == url;
    }).parent().addClass('active');
});


function bytes2Str(arr) {
    var str = "";
    for (var i = 0; i < arr.length; i++) {
        var tmp = arr[i].toString(16);
        if (tmp.length == 1) {
            tmp = "0" + tmp;
        }
        str += tmp;
    }
    return str;
}