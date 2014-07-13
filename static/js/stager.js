/**
 * Stager Polling Script
 * Copyright 2014 - go-stager
 */

(function poll() {

    var request = new XMLHttpRequest();
    request.onreadystatechange = function() {

        // Check to see if the request has completed
        if(request.readyState === 4) {

            // Anything other than a 200 status code is an error
            if(request.status !== 200) {

                // Display the error
                var status = document.getElementsByClassName('status')[0];
                status.className = 'status error';
                status.textContent = request.responseText || 'Something went bad.';

            } else {

                // If the instance is ready, refresh
                if(request.responseText == 'true')
                    location.reload();
                // ...otherwise, check again in two seconds
                else
                    setTimeout(poll, 2000);
            }
        }
    };

    request.open('GET', '/_stager/api/ready', true);
    request.send(null);

})();
