
var host = ""

function panel(body, header, footer) {
	var panel = $('<div class="panel panel-default">')
	if (header != null) {
		panel.append($('<div class="panel-heading">').append(header))
	}
	panel.append($('<div class="panel-body">').append(body))
	if (footer != null) {
			panel.append($('<div class="panel-footer">').append(footer))
	}
	
	return panel
}

function link(href, child, blank) {
	var url = $('<a>').attr('href', href).append(child)
	if (blank) {
		url.attr('target', '_blank')
	}
	return url
}

function row(child) {
	return $('<div class="row">').append(child)
}

function col(n, child) {
	return $('<div class="col-md-' + n + '">').append(child)
}

function pagination(current, total, href) {
	var pg = $('<ul class="pager">')
	if (current > 0) {
		pg.append($('<li class="pull-left">').append(link(href + (current - 1), '上一页')))
	}
	
	if (current + 1 < total) {
			pg.append($('<li class="pull-right">').append(link(href + (current + 1), '下一页')))
	}
	
	return row(pg)
}

function getResponse(data) {
	err = data['error']
	if (err['error_id'] != 0)
		return null
		
	return data['response_data']
}