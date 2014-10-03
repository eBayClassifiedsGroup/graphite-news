var gn = {};

gn.JsonPullInterval = 0;
gn.GraphiteURL = '';
gn.getConfig = function() {
	var jqxhr = $.getJSON( "/config/", function() {
	})
	.done(function() {
		data = jqxhr.responseJSON;
		gn.JsonPullInterval = data.JsonPullInterval;
		gn.GraphiteURL = data.GraphiteURL;
		gn.AllowDsDeletes = data.AllowDsDeletes;
		gn.Version = data.Version;
		gn.CompileTime = data.CompileTime;

		// New config is read in, but the interval is attached to the
		// timer and wont be updated dynamically. So depending on the 
		// current state (active/inactive) we need to cycle that
		if (gn.timer === undefined) {
			// not running, noop
		} else {
			gn.toggle();
			gn.toggle();
		}

		// Update Version number
		$('#gn-version').text(gn.Version);
	});
}

function template(row, dss) {
	row.find('.item_name').text(dss.Name);
	row.find('.item_date').html("<abbr class='timeago' title='"+dss.Create_date+"'>"+dss.Create_date+"</abbr>");
	row.find('.item_options').text(dss.Params);
	return row;
}

function endsWith(str, suffix) {
	return str.indexOf(suffix, str.length - suffix.length) !== -1;
}

gn.RemoveEnabledHTML = function() {
	if (!gn.AllowDsDeletes) {
		return ' disabled '
	} else {
		return ''
	}
}

gn.GraphiteImg = function(dsname, dsdate, width, derivative, editlink) {
	// If derivative is 'true', encapsulate the DS with a derivative function
	// in graphite. If none is given, determine if DS ends in '.*count', in 
	// which case we'll turn it on anyways.
	if (derivative === undefined) {
		if (endsWith(dsname, ".count")) {
			derivative = true
		} else {
			derivative = false
		}
	} else {
		derivative = false
	}

	if ((width < 50) || (width === undefined)) { width = 800 /* some default */ }
	if (editlink === undefined) { editlink = false }
	if (!editlink) { render = "render/" } else { render = "" }

	tmp = "";
	tmp = tmp + gn.GraphiteURL
	tmp = tmp + "/" + render + "?width=" +Math.floor(width)
	tmp = tmp + "&target=cactiStyle("
	if(derivative) { tmp = tmp + "perSecond(" }
	tmp = tmp + escape(dsname)
	if(derivative) { tmp = tmp + ")" }
	tmp = tmp + ",'si')"
	tmp = tmp + "&lineMode=connected&areaAlpha=0.15&areaMode=all"

	return tmp
}

// Update the set of data sources in the table, filter out
// the ones that we already have based on DS name.
gn.updateDs = function() {
	var jqxhr = $.getJSON( "/json/", function() {
		// initial success on calling getJSON
	})
	.done(function() {
		gn.serverActive();
		data = jqxhr.responseJSON;
		$("#dscount").text(data.length);
		$.map(data, function(el) {
			// map all the data onto the HTML table
			// if el.Name already exists, just skip it
			if ($('span').filter(
				function (index) { return $(this).text() == el.Name; }
			).length == 0)
		{
			var newRow = $('#cart .template').clone().removeClass('template');
			template(newRow, el)
			.prependTo('#cart')
			.fadeIn()

			// Add hover/click event listners to the table, because
			// I don't know jQuery well enough to solve this in a different
			// way (prependTo call looses those listners)
			.hover(function() {
				$(this).addClass('gnhover');
			}, function() {
				$(this).removeClass('gnhover');
			})
			// on click, add a row underneath with the graph(s)
			.click(function() {
				// If we have a .timeseries row underneath us, all we need
				// to do is remove them and we're done (allows for click, click once more
				// behaviour on normal ds td's)
				// TODO, don't know how to do yet!

				// find any and all tr's with class timeseries and remove them (want to keep UI
				// simple for now and only have one open at a time
				$('#cart .timeseries').remove();

				// now copy a template and add that behind the row that was just clicked
				var tmp = $('#cart .templateds').clone()
				.removeClass('templateds')
				.addClass('timeseries');
				tmp.find('td:first')
				.html(
					"<div class='timeseriescontainer'>"
					+"  <img class='img-rounded' src=\""+
						gn.GraphiteImg(
							$(this).find("td:first").text(),
					"none",
					Math.floor($(this).width() * 0.9))+"\">"
					+ '<span class="tsbtntoolbar">'
					+ '  <div class="btn-group btn-group-sm">'
					+ '    <a href="'+gn.GraphiteImg($(this).find("td:first").text(),"none",undefined,undefined,true)+'" type="button" class="btn btn-default">Edit</button>'
					+ '    <a id="btnRemove" href="" type="button" class="btn btn-default'+ gn.RemoveEnabledHTML() +'">Remove</button>'
					+ '  <div>'
					+ '</span>'
					+ '</div>'
				)
				.end()
				.click( function(){ $(this).remove() })
				.insertAfter('#cart .gnhover').fadeIn();

				// if we allow deletes, attach a handler
				if (gn.AllowDsDeletes) {
					dsName = $(this).find("td:first").text();
					$("#btnRemove").click(function() {

						$.post("/delete/", { datasourcename: dsName } )
						.done(function(data) {
							$.growl({
								title: '<strong>DELETING DATA SOURCE</strong><br/><i>'+dsName+'</i><br/>',
								message: 'Deleting the data source worked. If it re-appears it<br/>is because the data source provider kept on sending metrics.'
							},{
								type: 'success', delay: 5000, mouse_over: 'pauze', offset: 75
							});
							// remove the row from the table
							$('tr:has(td:contains("'+dsName+'"))').remove()
						})
						.fail(function(data) {
							$.growl({
								title: '<strong>DELETING DATA SOURCE</strong><br/><i>'+dsName+'</i><br/>',
								message: 'Deleting the data source did not work<br/>Possible reasons</br><ul><li>Perhaps the item does not exist</li><li>Permissions problem server-side</li></ul>'
							},{
								type: 'danger', delay: 5000, mouse_over: 'pauze', offset: 75
							});
						})
						;

						// remove all opened timeseries
						$('#cart .timeseries').remove()
						return false
					});
				}
			});
			// update "Date" column with humanized timestamps
			jQuery("abbr.timeago").timeago();
		}
		});
	})
	.fail(function() {
		gn.serverInactive();
		console.log( "ERROR: Could not perform getJSON request. Server down?" );
	})
}

// functions to signal the status of connectivity to backend server
gn.serverActive = function() {
	$("#servercon").addClass('label-success');
	$("#servercon").removeClass('label-danger');
	$("#cart").removeClass('transparent');
}
gn.serverInactive = function() {
	$("#servercon").removeClass('label-success');
	$("#servercon").addClass('label-danger');
	$("#cart").addClass('transparent');
}

// Starting, Stopping and toggling our requests to the
// server.
gn.start = function() {
	if(!gn.timer) {
		gn.timer = setInterval(function() {gn.updateDs();}, gn.JsonPullInterval);
		$("#hideButton").text('Pause')
		$("#hideButton").toggleClass('btn-danger');
		$("#hideButton").toggleClass('btn-success');
	}
}
gn.stop = function() {
	clearInterval(gn.timer);
	gn.timer = undefined;
	$("#hideButton").text('Activate')
	$("#hideButton").toggleClass('btn-danger');
	$("#hideButton").toggleClass('btn-success');
}
gn.toggle = function() {
	if (gn.timer === undefined) {
		gn.start();
	} else {
		gn.stop();
	}
}

var init = false;
$(document).ready(function() {
	if (!init){
		init = true;

		gn.getConfig()
		gn.start()

		$("#hideButton").click( function() {
			gn.toggle();
		});
		gn.updateDs()
		setInterval(function() {gn.getConfig();}, /* 1 minute */ 1*60*1000);
	}
});
