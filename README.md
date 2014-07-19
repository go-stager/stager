![Stager Logo](http://go-stager.github.io/img/logo.png)

Stager takes the pain out of running many development server instances.
It acts as a proxy and manager for a number of instances that run behind it,
starting and cleaning them up as needed. The instances are selected simply, by using
a domain name prefix to choose between them.

This can be used in a number of potential situations:
* Staging instances for developers running tests
* Testing links from ticket / issue trackers
* Continuous integration servers / buildbots needing access to a URL running a specific version of code. (think API testers)

### Features

Stager offers the following features:

 - Flexibility in what gets staged (branches, commits, etc.)
 - The ability to run more than one instance simultaneously
 - Automatic cleanup of instances that are no longer used
