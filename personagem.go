// personagem.go - Funções para movimentação e ações do personagem
package main

// Atualiza a posição do personagem com base na tecla pressionada (WASD)
func personagemMover(tecla rune, jogo *Jogo) {
	dx, dy := 0, 0
	switch tecla {
	case 'w', 'W':
		dy = -1 // Move para cima
	case 'a', 'A':
		dx = -1 // Move para a esquerda
	case 's', 'S':
		dy = 1 // Move para baixo
	case 'd', 'D':
		dx = 1 // Move para a direita
	}

	nx, ny := jogo.PosX+dx, jogo.PosY+dy
	// Verifica se o movimento é permitido e realiza a movimentação
	if jogoPodeMoverPara(jogo, nx, ny) {
		jogoMoverElemento(jogo, jogo.PosX, jogo.PosY, dx, dy)
		jogo.PosX, jogo.PosY = nx, ny
	}
}
