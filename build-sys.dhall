let Prelude = https://prelude.dhall-lang.org/package.dhall
let Action =
	< Copy :
		{ src : Text
		, dst : Text
		}
	| Run :
		{ cmd : Text
		, stdout : Optional Text
		}
	| Env :
		{ key : Text
		, val : Text
		}
	>
let Config =
	{ name : Text
	, actions : List Action
	}
let GoAction =
	{ type : Natural
	, src : Text
	, dst : Text
	}
let action2go
	: Action -> GoAction
	= \(action : Action) ->
		merge
		{ Copy = \(x : {dst : Text, src : Text}) ->
			{ type = 0
			, src = x.src
			, dst = x.dst
			}
		, Run = \(x : {cmd : Text, stdout : Optional Text}) ->
			{ type = 1
			, src = x.cmd
			, dst = Prelude.Optional.default Text "" x.stdout
			}
		, Env = \(x : {key : Text, val : Text}) ->
			{ type = 2
			, src = x.key
			, dst = x.val
			}
		} action
let GoConfig =
	{ name : Text
	, actions : List GoAction
	}
let config2go = \(config : Config) ->
	{ name = config.name
	, actions = Prelude.List.map Action GoAction action2go config.actions
	}
in Prelude.List.map Config GoConfig config2go
	[
		{ name = "hello"
		, actions =
			[ Action.Copy { src = "main.go", dst = "main.go" }
			, Action.Run { cmd = "echo hello", stdout = Some "hello.txt" }
			, Action.Run { cmd = "date", stdout = Some "date.txt" }
			, Action.Run { cmd = "pwd", stdout = None Text}
			, Action.Run { cmd = "rm -f output/latest", stdout = None Text}
			, Action.Run
				{ cmd = "echo ln -s $BUILDSYS_DIR $BUILDSYS_PARENT_DIR/latest"
				, stdout = None Text
				}
			, Action.Run
				{ cmd = "ln -s $BUILDSYS_DIR $BUILDSYS_PARENT_DIR/latest"
				, stdout = None Text
				}
			]
		}
	]

