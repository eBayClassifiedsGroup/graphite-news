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

    // New config is read in, but the interval is attached to the
    // timer and wont be updated dynamically. So depending on the 
    // current state (active/inactive) we need to cycle that
    if (gn.timer === undefined) {
      // not running, noop
    } else {
      gn.toggle();
      gn.toggle();
    }
  });
}

function template(row, dss) {
  row.find('.item_name').text(dss.Name);
  row.find('.item_date').text(dss.Create_date);
  row.find('.item_options').text(dss.Params);
  return row;
}

gn.GraphiteImg = function(dsname, dsdate, width) {
	if (width < 50) { width = 800 /* some default */ }
	tmp = "";
	tmp = tmp + gn.GraphiteURL
	tmp = tmp + "/render/?width=" +Math.floor(width)
	tmp = tmp + "&target=cactiStyle("
	tmp = tmp + escape(dsname)+ ",'si')"
	tmp = tmp + "&lineMode=connected"

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
              .html("<img src=\""+
                gn.GraphiteImg(
                  $(this).find("td:first").text(),
                  "none",
                  Math.floor($(this).width() * 0.9))+"\">")
              .end()
              .click( function(){ $(this).remove() })
              .insertAfter('#cart .gnhover').fadeIn();
	  });
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
