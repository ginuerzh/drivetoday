
var host = ""

var accessToken = "guest:0123456789"

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


function createUser(data, withModal) {
	var user = $('<div class="row">')
	var profile = $('<img class="user-profile img-circle" src="/images/1.gif">')
	if (data['profile_image'].length > 0) profile.attr('src', data['profile_image'])

	var nick = data['nikename']
	if (nick.indexOf('weibo_') == 0) {
		nick = nick.substring(6)
	}

	var nickname = $('<span>').append(statusIcon(data['online']), nick)

	var regtime = $('<span class="reg-time">').text(data['register_time'].substring(2, 16))
	var location = $('<span>').text(data['location'])
	//var about = $('<span>').text(data['about'])

	var view = $('<span class="stat pull-right">')
	var thumb = $('<span class="stat pull-right">')
	var review = $('<span class="stat pull-right">')

	if (withModal) {
		view.append($('<a href="#" class="view-modal-btn" data-toggle="modal" data-target="#viewModal">').attr('id', data['userid']).
							append(data['view_count'] + ' ', glyphicon('eye-open')))
		thumb.append($('<a href="#" class="thumb-modal-btn" data-toggle="modal" data-target="#thumbModal">').attr('id', data['userid']).
							append(data['thumb_count'] + ' ', glyphicon('thumbs-up')))
		review.append($('<a href="#" class="comment-modal-btn" data-toggle="modal" data-target="#commentModal">').attr('id', data['userid']).
							append(data['review_count'] + ' ', glyphicon('comment')))
	} else {
		view.append(data['view_count'] + ' ', glyphicon('eye-open'))
		thumb.append(data['thumb_count'] + ' ', glyphicon('thumbs-up'))
		review.append(data['review_count'] + ' ', glyphicon('comment'))
	}

	user.append(col(2, profile))
	user.append(col(6, link(host+'/user.html?uid=' + data['userid'], nickname)).
			append('<br>', regtime).
			append('<br>', location))
	user.append(col(4).append(row(col(12, view)), row(col(12, thumb)), row(col(12, review))))

	return user
}


function createReview(data) {
	var review = $('<div class="row">')
	var profile = $('<img class="profile img-circle" src="/images/1.gif">')
	var nickname = $('<span>')
	if (data['review_author'].indexOf('guest:') == 0) {
		nickname.text('匿名用户')
	} else {
		nickname.text(data['review_author'])
		nickname = link(host+'/user.html?uid=' + data['review_author'], nickname)
	}
	var reviewtime = $('<span class="review-time">').text(data['time'].substring(2, 16))

	review.append(col(1, profile))
	var userlink = link(host+'/user.html?uid=' + data['review_author'], nickname)
	review.append(col(3, userlink).append('<br>', reviewtime))
	review.append(col(6, data['message']))

	var thumb = $('<span class="stat thumb-count pull-right">').append(data['thumb_count'] + ' ', glyphicon('thumbs-up'))
	review.append(col(2, thumb))

	$.getJSON(host + "/1/user/getInfo?userid=" + data['review_author'], function(data){
		var userinfo = getResponse(data)
		if (userinfo == null) return

		if (userinfo['profile_image'].length > 0) profile.attr('src', userinfo['profile_image'])

		var nick = userinfo['nikename']
		if (nick.indexOf('weibo_') == 0) {
			nick = nick.substring(6)
		}
		nickname.text(nick).prepend(statusIcon(userinfo['online']))
	})

	return review
}

function createArticle(data, withModal) {
	var article = $('<div class="row">')
	var detail = col(12)
	var title = $('<div class="row">')
	var info = $('<div class="row">')

	title.append(col(10, link("/article.html?id=" + data['article_id'], $('<span class="h4">').text(data['title']))))
	title.append(col(2, $('<img class="thumbnail">').attr('src', data['first_image'])))
	detail.append(title)

	info.append(col(2, link(data['src_link'], data['source'], true)))
	info.append(col(3, data['publish_time'].substring(2, 16)))

	var view = $('<span class="stat">')
	var thumb = $('<span class="stat">')
	var review = $('<span class="stat">')

	if (withModal) {
		view.append($('<a href="#" class="view-modal-btn" data-toggle="modal" data-target="#viewModal">').attr('id', data['article_id']).
						append(data['view_count'] + ' ', glyphicon('eye-open')))
		thumb.append($('<a href="#" class="thumb-modal-btn" data-toggle="modal" data-target="#thumbModal">').attr('id', data['article_id']).
						append(data['thumb_count'] + ' ', glyphicon('thumbs-up')))
		review.append($('<a href="#" class="comment-modal-btn" data-toggle="modal" data-target="#commentModal">').attr('id', data['article_id']).
						append(data['comment_count'] + ' ', glyphicon('comment')))
	} else {
		view.append(data['view_count'] + ' ', glyphicon('eye-open'))
		thumb.append(data['thumb_count'] + ' ', glyphicon('thumbs-up'))
		review.append(data['comment_count'] + ' ', glyphicon('comment'))
	}

	info.append(col(2, view))
	info.append(col(2, thumb))
	info.append(col(2, review))
	detail.append(info)

	article.append(detail)

	return $('<div class="row">').append(col(12, panel(article)))
}

function getResponse(data) {
	err = data['error']
	if (err['error_id'] != 0)
		return null

	return data['response_data']
}

function getReviews(data) {
 	var resp = getResponse(data)
 	if (resp == null) 	return null
 	else 				return resp['reviews']
}
