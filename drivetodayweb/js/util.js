
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

function glyphicon(name) {
	return $('<span class="glyphicon glyphicon-' + name + '">')
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

function modal(id, title, body, footer) {
	var modal = $('<div class="modal fade" tabindex="-1" role="dialog" aria-labelledby="myModalLabel" aria-hidden="true">').attr('id', id)
	var modal_dialog = $('<div class="modal-dialog">')
	var modal_content = $('<div class="modal-content">')
	
	var modal_header = $('<div class="modal-header">')
	var close_btn = $('<button type="button" class="close" data-dismiss="modal" aria-hidden="true">').text('&times;')
	var modal_title = $('<h4 class="modal-title" id="myModalLabel">').text(title)
	modal_header.append(close_btn, modal_title)
	
	var modal_body = $('<div class="modal-body">').append(body)
	var modal_footer = $('<div class="modal-footer">').append(footer)
	
	modal_content.append(modal_header, modal_body, modal_footer)
	modal_dialog.append(modal_content)
	modal.append(modal_dialog)
	
	return modal
}


function statusIcon(online) {
	if (online)
		return $('<span class="glyphicon glyphicon-user online">')
	else
		return $('<span class="glyphicon glyphicon-user offline">')
}


function createUser(data) {
	var user = $('<div class="row">')
	var profile = $('<img class="user-profile img-circle" src="/images/1.gif">')
	if (data['profile_image'].length > 0) profile.attr('src', data['profile_image'])	
	
	var nick = data['nikename']
	if (nick.indexOf('weibo_') == 0) {
		nick = nick.substring(6)	
	}	
	
	var nickname = $('<span>').append(statusIcon(data['online']), nick)
	
	var regtime = $('<span class="reg-time">').text(data['register_time'])
	var location = $('<span>').text(data['location'])
	//var about = $('<span>').text(data['about'])
	
	var view = $('<span class="stat pull-right">').append(data['view_count'] + ' ', glyphicon('eye-open'))
	var thumb = $('<span class="stat pull-right">').append(data['thumb_count'] + ' ', glyphicon('thumbs-up'))
	var review = $('<span class="stat pull-right">').append(data['review_count'] + ' ', glyphicon('comment'))
	
	user.append(col(2, profile))
	user.append(col(6, link(host+'/user.html?uid=' + data['userid'], nickname)).
			append('<br>', regtime).
			append('<br>', location))
	user.append(col(4).append(view, $('<br>'), thumb, $('<br>'), review))
	
	return user
}


function getResponse(data) {
	err = data['error']
	if (err['error_id'] != 0)
		return null
		
	return data['response_data']
}