var gn = {};
gn.refresh_ms = 1500;

function template(row, dss) {
  row.find('.item_name').text(dss.Name);
  row.find('.item_date').text(dss.Create_date);
  row.find('.item_options').text(dss.Params);
  return row;
}



gn.updateDs = function() {
	$.getJSON(
			'/json/',
			function(data) {
			//console.table(data);
			$("#dscount").text(data.length);

			$.map(data, function(el) {
				// map all the data onto the HTML table
				// if el.Name already exists, just skip it
				var newRow = $('#cart .template').clone().removeClass('template');
				template(newRow, el)
				.prependTo('#cart')
				.fadeIn();
				});
			});
}

gn.start = function() {
	if(!gn.timer) {
	gn.timer = setInterval(function() {gn.updateDs();}, gn.refresh_ms);
	$("#hideButton").text('Disconnect')
  }
}
gn.stop = function() {
	clearInterval(gn.timer);
	gn.timer = undefined;
	$("#hideButton").text('Reconnect')
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
	}
});
