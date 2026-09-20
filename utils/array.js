/** 搜索数组*/
export function search_object(array, field, keywords) {
    if (typeof array !== 'object') {
        return false;
    } else {
        var found = [];
        for (var i = 0; array.length > i; i++) {
            if (typeof field == 'object') {
                for (var j = 0; field.length > j; j++) {
                    var field_str = field[j];
                    var str = array[i][field_str];
                    // 只需要匹配到一个即可
                    if (str.indexOf(keywords) != -1) {
                        found.push(array[i]);
                        break;
                    }
                }
            } else {
                var str = array[i][field];
                if (str.indexOf(keywords) != -1) {
                    found.push(array[i]);
                }
            }
        }
        return found;
    }
}

/** 数组转化为字符串*/
export function arrayToString(arr, field, isTrans) {
    isTrans = typeof isTrans !== 'undefined' ? isTrans : false;
    var idr = [];
    for (var i = 0; i < arr.length; i++) {
        idr.push(arr[i][field]);
    }
    if (isTrans === true) {
        return idr.join(",");
    } else {
        return idr;
    }
}


/** 删除数组中某一个值*/
export function delArrValue(arr, field, value) {
    for (var i = 0; i < arr.length; i++) {
        if (arr[i][field] == value) {
            arr.splice(i, 1);
        }
    }
    return arr;
}

/** 修改二位数组某个值*/
export function editArrValue(arr, field, value) {
    for (var i = 0; i < arr.length; i++) {
        arr[i]['disabled'] = false;
        if (arr[i][field] == value) {
            arr[i]['disabled'] = true;
        }
    }
    return arr;
}