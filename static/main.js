$("#route").click(function() {
	$("#loading").css("display", "block");
	$("#result").html("")
	var cities = {
		startCity: $("#startCity").val().trim().toUpperCase(),
		startState: $("#startState").val(),
		endCity: $("#endCity").val().trim().toUpperCase(),
		endState: $("#endState").val()
	};
	$.get("/route/", cities, function(result) {
		$("#loading").css("display", "none");
		if (result === "Bad start") {
			$("#result").text("Unknown start city.");
			return;
		}
		if (result === "Bad end") {
			$("#result").text("Unknown end city.");
			return;
		}
		$("#result").append("<ul></ul>");
		var list = $("#result ul");
		list.append("<li>Start in " + name(result[0]) + ".</li>");
		for (var i = 1; i < result.length; i++) {
			var from = result[i-1];
			var to = result[i];
			if (from.City === to.City) {
				list.append("<li>Take the wormhole between " + name(from) + " and " + name(to) + ".</li>");
			} else {
				list.append("<li>Drive from " + name(from) + " to " + name(to) + ":<br>"
					+ "<iframe width=600 height=480 "
					+ "src=\"https://www.google.com/maps/embed/v1/directions"
					+ "?origin=" + encodeURIComponent(name(from))
					+ "&destination=" + encodeURIComponent(name(to))
					+ "&key=AIzaSyC7kADz-9VYGr2U3jgkDGDu5t4XgO2bQPs"
					+ "\"></iframe></li>"
				);
			}
		}
		list.append("<li>You have reached your destination.");
	}, "json");
});

function name(place) {
	return place.City + ", " + place.State;
}
