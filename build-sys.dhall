let Prelude = https://prelude.dhall-lang.org/v15.0.0/package.dhall
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
	{ actions : List Action
	, dir : Text
	}
-- no unions
let GoAction =
	{ `type` : Natural
	, src : Text
	, dst : Text
	}
let action2go
	: Action -> GoAction
	= \(action : Action) ->
		merge
		{ Copy = \(x : {dst : Text, src : Text}) ->
			{ `type` = 0
			, src = x.src
			, dst = x.dst
			}
		, Run = \(x : {cmd : Text, stdout : Optional Text}) ->
			{ `type` = 1
			, src = x.cmd
			, dst = Prelude.Optional.default Text "" x.stdout
			}
		, Env = \(x : {key : Text, val : Text}) ->
			{ `type` = 2
			, src = x.key
			, dst = x.val
			}
		} action
let GoConfig =
	{ actions : List GoAction
	, dir : Text
	}
let config2go = \(config : Config) ->
	{ actions = Prelude.List.map Action GoAction action2go config.actions
	, dir = config.dir
	}
in config2go
	{ actions =
		[ Action.Copy { src = "main.go", dst = "main.go" }
		, Action.Run { cmd = "echo hello", stdout = Some "hello.txt" }
		]
	, dir = "output/hello"
	}

