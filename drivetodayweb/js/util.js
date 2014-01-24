
var host = "http://localhost:8080"

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

function col(n, child) {
	return $('<div class="col-md-' + n + '">').append(child)
}