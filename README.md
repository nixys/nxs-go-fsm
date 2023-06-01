# Nixys Flow State Machine library

Nixys Flow State Machine is the library for Golang implements [Finite-state machine](https://en.wikipedia.org/wiki/Finite-state_machine) principles for the data flow.

## Introduction

An `io.Reader` (`fsm.Reader` further in the text) you will get after initialize the library represents a finite-state machine. The `fsm.Reader` it's a wrap on specified `io.Reader` (it may be any `io.Reader`, e.g. `os.Stdin`, `os.File` or even other `fsm.Reader`) with described states. Using `fsm.Reader` you may read the data from some source with the required substitutions on the flow.

At any point of time `fsm.Reader` is in one of specified states. Every state has a set of next states. When `trigger` (token) for one of the next state is encountered in the data flow and all conditions (described in `switch`) are match, `data handler` for this next state will be called and machine will be switched to.

## Import

```go
import fsm "github.com/nixys/nxs-go-fsm"
```

## Description of usage

### Initialize

You need to initialize this library via `Init(r io.Reader, desc fsm.Description)` function before using. After that you will get a `fsm.Reader` reader. Via this reader you'll get a modified data in accordance with described states. To describe a Nixys Flow State Machine you need to determine following options:
- `r`: any `io.Reader` you wish to read data from
- `desc`: struct to describe a states and contexts. See below for details.

#### Type `fsm.StateName`

Data type to specify state name. Based on data type `string`.

#### Struct `fsm.Description`

| Field     | Type                        | Description                                                           |
|-----------|-----------------------------|-----------------------------------------------------------------------|
| Ctx       | context.Context             | Go context                                                            |
| UserCtx   | any                         | User context. Used in `data handlers` to operate with any user's data |
| States    | map[fsm.StateName]fsm.State | Machine states description                                            |
| InitState | fsm.StateName               | State used by the machine as an initial state                         |

#### Struct `fsm.State`

| Field      | Type            | Description                                                                        |
|------------|-----------------|------------------------------------------------------------------------------------|
| NextStates | []fsm.NextState | Set of next states. May be empty, in this case machine will never change its state |

#### Struct `fsm.NextState`

| Field       | Type                                                     | Description                                |
|-------------|----------------------------------------------------------|--------------------------------------------|
| Name        | fsm.StateName                                            | Name of the next state                     |
| Switch      | fsm.Switch                                               | Conditions to select the next state        |
| DataHandler | func(userCtx any, data, trigger []byte) ([]byte, error)  | Function to be called (if not nil) if switch conditions are match. Within the function you are able to use `user context` specified at init, `data` (all bytes read from the stream since the machine was put into this state) and `trigger` (token from the data flow that switched machine to next state). Function returns a bytes to be write into output buffer           |

#### Struct `fsm.Switch`

| Field      | Type            | Description                                |
|------------|-----------------|--------------------------------------------|
| Trigger    | []byte          | Bytes sequence in a data flow that switches machine in the specified state. To be trigger is considered matched a conditions described in lines below in this table must be met                          |
| Delimiters | fsm.Delimiters  | Right and left trigger delimiters. The trigger is considered matched if the specified delimiters are present around the trigger. Delimiters can be empty, in this case trigger may has any bytes arounded    |
| Escape     | bool            | Set trigger sensitive to escape character '\'. If true a trigger is considered matched only if preceding character is not a '\' (or even number of this character). Works in conjunction with delimiters    |

#### Struct `fsm.Delimiters`

| Field | Type   | Description                                                         |
|-------|--------|---------------------------------------------------------------------|
| L     | []byte | Left delimiters of trigger. It's a set of bytes. When left delimiters are set, trigger considered matched if one preceding byte before trigger present in a left delimiter set. Works in conjunction with right delimiters and escape |
| R     | []byte | Right delimiters of trigger. It's a set of bytes. When right delimiters are set, trigger considered matched if one following byte after trigger present in a right delimiter set. Works in conjunction with left delimiters and escape |

## Example

In the example below string `somePgSQLDummyPlainDump` used as an input data flow. In this flow all columns excluding last will changed to the `000` bytes sequence and the last one to `abc`.

```go
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	fsm "github.com/nixys/nxs-go-fsm"
)

const somePgSQLDummyPlainDump = `_Some previous data_

---
--- Columns separated by the space!
---

COPY public.names (id, name) FROM stdin;
12 alice
34 bob
\.

_Some following data_

COPY public.comments (id, comment) FROM stdin;
45 foo
78 bar
\.

_Some other following data_
`

var (
	stateCopySearch  = fsm.StateName("copy search")
	stateCopyTail    = fsm.StateName("copy tail")
	stateTableValues = fsm.StateName("table values")
)

func main() {

	r := strings.NewReader(somePgSQLDummyPlainDump)

	fsmR := fsm.Init(
		r,
		fsm.Description{
			Ctx:       context.TODO(),
			UserCtx:   nil,
			InitState: stateCopySearch,
			States: map[fsm.StateName]fsm.State{

				stateCopySearch: {
					NextStates: []fsm.NextState{
						{
							Name: stateCopyTail,
							Switch: fsm.Switch{
								Trigger: []byte("COPY"),
								Delimiters: fsm.Delimiters{
									L: []byte{'\n'},
									R: []byte{' '},
								},
							},
							DataHandler: nil,
						},
					},
				},
				stateCopyTail: {
					NextStates: []fsm.NextState{
						{
							Name: stateTableValues,
							Switch: fsm.Switch{
								Trigger: []byte(";\n"),
							},
							DataHandler: nil,
						},
					},
				},
				stateTableValues: {
					NextStates: []fsm.NextState{
						{
							Name: stateCopySearch,
							Switch: fsm.Switch{
								Trigger: []byte("\\."),
								Delimiters: fsm.Delimiters{
									L: []byte{'\n'},
									R: []byte{'\n'},
								},
							},
							DataHandler: nil,
						},
						{
							Name: stateTableValues,
							Switch: fsm.Switch{
								Trigger: []byte{' '},
							},
							DataHandler: dhTableValueColumn1,
						},
						{
							Name: stateTableValues,
							Switch: fsm.Switch{
								Trigger: []byte{'\n'},
							},
							DataHandler: dhTableValueColumn2,
						},
					},
				},
			},
		},
	)

	_, err := io.Copy(os.Stdout, fsmR)
	if err != nil {
		fmt.Fprintf(os.Stderr, "copy error: %s", err)
	}
}

func dhTableValueColumn1(usrCtx any, data, trigger []byte) ([]byte, error) {
	return append([]byte("000"), trigger...), nil
}

func dhTableValueColumn2(usrCtx any, data, trigger []byte) ([]byte, error) {
	return append([]byte("abc"), trigger...), nil
}
```

Run:

```
go run main.go
```

Output:

```
_Some previous data_

---
--- Columns separated by the space!
---

COPY public.names (id, name) FROM stdin;
000 abc
000 abc
\.

_Some following data_

COPY public.comments (id, comment) FROM stdin;
000 abc
000 abc
\.

_Some other following data_
```

**For more examples see apps based on this library: _coming soon_**
