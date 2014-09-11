var gn = {};
gn.refresh_ms = 5000;

function template(row, dss) {
  row.find('.item_name').text(dss.Name);
  row.find('.item_date').text(dss.Create_date);
  row.find('.item_options').text(dss.Params);
  return row;
}

// Update the set of data sources in the table, filter out
// the ones that we already have based on DS name.
gn.updateDs = function() {
	$.getJSON(
			'/json/',
			function(data) {
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
						.fadeIn();
					}
				});
			});
}

gn.start = function() {
	if(!gn.timer) {
	gn.timer = setInterval(function() {gn.updateDs();}, gn.refresh_ms);
	$("#hideButton").text('Disconnect')
	$("#hideButton").toggleClass('btn-danger');
	$("#hideButton").toggleClass('btn-success');
  }
}
gn.stop = function() {
	clearInterval(gn.timer);
	gn.timer = undefined;
	$("#hideButton").text('Reconnect')
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
		gn.start()
		$("#hideButton").click( function() {
			gn.toggle();
		});
		gn.updateDs()
	}
});
