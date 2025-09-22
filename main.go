package main

import (
	"os"
)

func main() {
	interfaceIniciar()
	defer interfaceFinalizar()

	mapaFile := "mapa.txt"
	if len(os.Args) > 1 {
		mapaFile = os.Args[1]
	}

	jogo := jogoNovo()
	if err := jogoCarregarMapa(mapaFile, &jogo); err != nil {
		panic(err)
	}

	// Canais principais
	chTeclado := make(chan EventoTeclado, 8)
	chCmd := make(chan Cmd, 64)

	// Sobe coordenador (dono do estado)
	coord := novoCoordenador(&jogo, chTeclado, chCmd)
	go coord.loop()

	// Captura de teclado concorrente
	go capturarTeclado(chTeclado)

	// --- Subscrição à posição do player para elementos ---
	chPosPlayerPortal := make(chan Ponto, 4)
	chPosPlayerSent := make(chan Ponto, 4)
	chPosPlayerTrap := make(chan Ponto, 4)
	chCmd <- CmdEscutarPosDoJogador{Ch: chPosPlayerPortal}
	chCmd <- CmdEscutarPosDoJogador{Ch: chPosPlayerSent}
	chCmd <- CmdEscutarPosDoJogador{Ch: chPosPlayerTrap}

	// --- Alavanca + Portal com timeout ---
	alavancaPos := Ponto{X: jogo.PosX + 2, Y: jogo.PosY}
	portalPos := Ponto{X: jogo.PosX + 4, Y: jogo.PosY}
	destino := Ponto{X: jogo.PosX + 10, Y: jogo.PosY + 2}

	chAbrirPortal := make(chan sinal, 1)

	// Agora a própria alavanca publica o sinal para abrir o portal
	iniciarAlavanca(alavancaPos.X, alavancaPos.Y, chCmd, chAbrirPortal)
	iniciarPortal(portalPos.X, portalPos.Y, destino, chCmd, chAbrirPortal, chPosPlayerPortal)

	// --- Sentinela com escuta múltipla ---
	wp := []Ponto{
		{X: jogo.PosX + 12, Y: jogo.PosY + 1},
		{X: jogo.PosX + 12, Y: jogo.PosY + 6},
	}
	iniciarSentinela("S1", wp[0], wp, 4, chCmd, chPosPlayerSent)

	// --- Armadilha oscilante (extra) ---
	trap := Ponto{X: jogo.PosX + 1, Y: jogo.PosY + 2}
	iniciarArmadilha(trap.X, trap.Y, 1500, 2000, chCmd, chPosPlayerTrap)

	// Bloqueia até o usuário sair (ESC)
	for ev := range chTeclado {
		if ev.Tipo == "sair" {
			break
		}
	}
	chCmd <- CmdQuit{}
}
