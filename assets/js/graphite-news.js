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
		console.log("asdfsafas"+ $(this));
		var tmp = $('#cart .templateds').clone().removeClass('templateds').addClass('timeseries');
		// empty text
		tmp.find('td').text('.');
		tmp.find('td:first').html("<img src=''"+$(this).width() * 0.8+"> Link"+$(this).find("td:first").text());
		tmp.insertAfter('#cart .gnhover').fadeIn();
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
  gn.timer = setInterval(function() {gn.updateDs();}, gn.refresh_ms);
  $("#hideButton").text('Pauze')
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

    gn.start()
    $("#hideButton").click( function() {
      gn.toggle();
    });
    gn.updateDs()
  }
});
