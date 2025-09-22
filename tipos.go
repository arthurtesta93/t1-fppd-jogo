package main

//objetos para facilitar a interface com o coordenador

type Ponto struct{ X, Y int }

// Comandos para o coordenador, enviados por elementos/teclado
type Cmd interface{}

type CmdQuit struct{}

type CmdStatus struct{ Texto string }

type CmdEscutarPosDoJogador struct{ Ch chan<- Ponto }

type CmdSetCelula struct {
	X, Y int
	Elem Elemento
}

type CmdTeleportarJogador struct {
	X, Y int // destino
}

type CmdTryMoveElemento struct {
	ID           string
	From, To     Ponto
	Elem         Elemento
	PodeSobrepor bool // se pode sobrepor (ex.: portal aberto, armadilha, etc.)
}

type CmdRegistrarInteragivel struct {
	X, Y int
	// coordenador enviará um "struct{}{}" aqui quando o jogador interagir nessa célula (ou adjacências)
	Ch chan<- struct{}
}
