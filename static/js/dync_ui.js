function dynamic_load(chain, ui_id, rst) {
    $("#" + ui_id).html("")
    var uis = $("#" + ui_id)
    var nat_list = $('<ul class="nav nav-tabs">')
    console.log("dynamic_load:", rst)
    rst.app = dataEncode(rst.app, "bytes2hex")
    $.each(rst.list, function (i, item) {
        var li = $('<li>').append(
            $('<a href="#uis_id_' + i + '" data-toggle="tab">').append($('<b>').append(item.info.title))
        )
        if (i == 0) {
            li.attr("class", "active")
        }
        if (item.info.is_view_ui == true) {
            li.css('background-color', '#00FFCC')
        }
        nat_list.append(li)
    });
    uis.append(nat_list)
    var tab_content = $('<div class="tab-content">')
    $.each(rst.list, function (i, item) {
        var it;
        var form = $('<form class="bs-example bs-example-form" role="form">');
        if (i == 0) {
            it = $('<div class="tab-pane active" id="uis_id_' + i + '">')
        } else {
            it = $('<div class="tab-pane" id="uis_id_' + i + '">')
        }
        var desc = $('<div class="input-group">').append(
            $('<span class="input-group-addon">Description</span>')
        )
        if (item.info.is_view_ui == true) {
            desc.append($('<span class="form-control">').append(
                'View UI,Only read and show data.'))
        } else {
            desc.append($('<span class="form-control">').append(
                'Update UI,run app'))
        }
        form.append(desc)
        if (item.info.is_view_ui != true) {
            form.append($('<div class="input-group">').append(
                $('<span class="input-group-addon">Energy</span>')
            ).append(
                $('<input type="number" class="form-control" name="energy" value="1">')
            ).append(
                $('<span class="input-group-addon">t9</span>')
            ))
        }

        $.each(item.items, function (j, sub_it) {
            var sit = $('<div class="input-group">').append(
                $('<span class="input-group-addon">').append(sub_it.key)
            )
            var vit = $('<input type="text" class="form-control">')
            if (item.info.is_view_ui == true)
                vit.attr("name", "name" + j);
            else
                vit.attr("name", sub_it.name);
            if (sub_it.value !== undefined) {
                vit.attr("value", sub_it.value)
            }
            if (sub_it.value_type !== undefined) {
                switch (sub_it.value_type) {
                    case "number":
                        vit.attr("type", "number")
                    case "bool":
                        vit.attr("type", "checkbox")
                    case "user":
                        vit.attr("value", rst.user)
                }
            }
            if (sub_it.description !== undefined) {
                vit.attr("placeholder", sub_it.description)
                vit.attr("title", sub_it.description)
            }
            if (sub_it.readonly == true) {
                vit.attr("readonly", "readonly")
            }
            if (sub_it.hide == true) {
                sit.attr("style", "display: none")
            }
            sit.append(vit)
            // if (sub_it.value_type !== undefined) {
            //     sit.append($('<span class="input-group-addon"">').append(sub_it.value_type))
            // } else {
            //     sit.append($('<span class="input-group-addon"">').append("string"))
            // }
            form.append(sit)
        });
        form.append('<br>')
        var btn = $('<button type="button" class="btn btn-success pull-right">Do</button>')
        form.append(btn)
        form.append($('<br>'))
        var rstEle = $('<div>')
        form.append(rstEle)
        btn.on('click', function () {
            console.log("click item:", i)
            if (item.info.is_view_ui == true) {
                var data = $(this).parent("form").serializeJSON();
                var key = "";
                $.each(item.items, function (j, sub_it) {
                    var type = sub_it.value_type;
                    // if (type === undefined || type == "") {
                    //     type = "str2hex"
                    // }
                    key += dataEncode(data["name" + j], type)
                });
                rstEle.html($('<h4>').append("Read Result:"));
                readFromDB(chain, rst.app, item.info.struct_name, (!item.info.is_log).toString(),
                    key, item.info.value_type, rstEle, item.view_items)
            } else {
                var data = $(this).parent("form").serializeJSON()
                var cost = 0;
                var prifix = "";
                if (item.info.cost !== undefined) {
                    cost = dataEncode(item.info.cost, "cost2int")
                }
                if (item.info.prefix_param !== undefined) {
                    prifix = item.info.prefix_param
                }
                rstEle.html($('<h4>').append("Run Result:"));
                var energy = parseInt(1000000000 * data.energy);
                delete data.energy;
                runApp(chain, rst.app, cost, prifix, data, rstEle, energy)
            }
        })
        tab_content.append(it.append(form))
    });
    uis.append(tab_content)
    // $("#" + ui_id).append(uis)
}

function runApp(chain, app, cost, prefix, data, element, energy) {
    var body = { "cost": cost, "energy": energy, "app_name": app, "param": prefix, "param_type": "json", "json_param": data };
    $.ajax({
        type: "POST",
        url: "/api/v1/" + chain + "/transaction/app/run",
        contentType: "application/json",
        data: JSON.stringify(body),
        dataType: "json",
        success: function (rst) {
            console.log(rst);
            element.append(
                $('<span class="label label-success">').append(
                    $('<a href="transaction.html?key=' + rst.key + '&chain=' + chain + '">').append("Transaction:" + rst.key)
                ))
        },
        error: function (XMLHttpRequest, textStatus, errorThrown) {
            console.log(XMLHttpRequest.responseText);
            console.log(textStatus);
            element.append("<span class=\"labellabel-danger\">errorcode:" + XMLHttpRequest.status +
                ". msg: " + XMLHttpRequest.responseText + "</span>");
        }
    });
}

function readFromDB(chain, app, struct, is_db, key, valueType, element, views) {
    $.ajax({
        type: "GET",
        url: "/api/v1/" + chain + "/data",
        // url: "test_data/data.json?v=1",
        data: { "app_name": app, "struct_name": struct, "is_db_data": is_db, "key": key },
        dataType: "json",
        success: function (rst) {
            if (rst.value === undefined || rst.value == "") {
                element.append("not found")
                return
            }
            rst.value = dataEncode(rst.value, valueType)
            rst.life = dataEncode(rst.life, "time2str")
            // console.log(JSON.stringify(rst, null, 2))
            element.append(
                $('<div class="input-group">').append(
                    $('<span class="input-group-addon">').append("Data Life:")
                ).append(
                    $('<span class="form-control">').append(rst.life)))
            $.each(views, function (j, sub_it) {
                var sit = $('<div class="input-group">').append(
                    $('<span class="input-group-addon">').append(sub_it.key)
                )
                var vit = $('<span class="form-control">').append(
                    dataToString(rst.value, sub_it.vk, sub_it.value_type))
                element.append(sit.append(vit))
            });
        },
        error: function (XMLHttpRequest, textStatus, errorThrown) {
            console.log(XMLHttpRequest.responseText);
            console.log(textStatus);
            element.append("<span class=\"label label-danger\">error code:" + XMLHttpRequest.status +
                ". msg: " + XMLHttpRequest.responseText + "</span>");
        }
    });
}

function dataToString(input, offset, typ) {
    offset = offset || "";
    sp = offset.split(".")
    for (i in sp) {
        if (sp[i] == "") {
            continue
        }
        input = input[sp[i]]
    }
    return dataEncode(input, typ)
}
