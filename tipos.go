// tipos.go
package main

type Ponto struct{ X, Y int }

type Cmd interface{}
type CmdQuit struct{}
type CmdStatus struct{ Texto string }

// ⬇️ AQUI: tecla como rune (e não string)
type CmdMovePlayer struct{ Tecla rune }

type CmdInteragir struct{}

type CmdSubscribePlayerPos struct{ Ch chan<- Ponto }

type CmdSetCell struct {
	X, Y int
	Elem Elemento
}
type CmdTeleportPlayer struct{ X, Y int }

type CmdTryMoveEntity struct {
	ID         string
	From, To   Ponto
	Elem       Elemento
	CanOverlap bool
}

type CmdRegisterInteractable struct {
	X, Y int
	Ch   chan<- struct{}
}
