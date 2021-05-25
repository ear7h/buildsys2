# buildsys 2

Another build system.

Reasons why this one is good:
* it's general
* cross platform
	* can be compiled for wherever Go runs, from wherever Go runs
* the core system is simple, thus more complex systems, like dhall
	can be interfaced simply.

TODO:
* more general argument system, using `[]string` instead of
	a constant number of fields
* better mechanism for run actions, `bash -c $SRC` seems like an unecessary
	dependency.
	* a better approach would be to make run match `exec.Command`,
		that is raw command running. Extra complexity of a scripting
		language (ex. pipes) can still be done by running `bash -c`,
		and made ergonomic with dhall.
* use CBOR as an intermediate format
* check the correctness and viability of the Go dhall implementation

