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

function getUrlParam(name) {
    var reg = new RegExp("(^|&)" + name + "=([^&]*)(&|$)");
    var r = window.location.search.substr(1).match(reg);
    if (r != null) return unescape(r[2]); 
    return "";
}

function fmoney(s) {
    // console.log("money:"+s)
    s = parseFloat((s + "").replace(/[^\d\.-]/g, "")).toFixed(10) + "";
    var l = s.split(".")[0].split("").reverse();
    var t = "";
    for (i = 0; i < l.length; i++) {
        t += l[i] + ((i + 1) % 3 == 0 && (i + 1) != l.length ? "," : "");
    }
    return t.split("").reverse().join("");
}

function getLinkString(path,chain,key){
    var out = ""
    var k = bytes2Str(key)
    if (k == "0000000000000000000000000000000000000000000000000000000000000000"){
        return "NULL"
    }
    out += "<a href=\""+path+"?key=" + k;
    out += "&chain="+chain + "\">" + k;
    return out
}
