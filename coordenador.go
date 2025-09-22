package main

import "time"

// Coordenador: único "dono" do estado do jogo.
// Recebe comandos de teclado e dos elementos autônomos e aplica no estado.
type coordenador struct {
	jogo          *Jogo
	chCmd         <-chan Cmd
	done          chan<- struct{}            // sinaliza término para main
	subsPos       []chan<- Ponto             // inscritos para posição do player
	interactables map[[2]int]chan<- struct{} // pontos interagíveis -> canal
}

func novoCoordenador(j *Jogo, chCmd <-chan Cmd, done chan<- struct{}) *coordenador {
	return &coordenador{
		jogo:          j,
		chCmd:         chCmd,
		done:          done,
		interactables: make(map[[2]int]chan<- struct{}),
	}
}

func (c *coordenador) publicarPosPlayer() {
	p := Ponto{c.jogo.PosX, c.jogo.PosY}
	for _, ch := range c.subsPos {
		select {
		case ch <- p:
		default:
		} // não bloquear o coordenador
	}
}

func (c *coordenador) desenhar() { interfaceDesenharJogo(c.jogo) }

func (c *coordenador) loop() {
	// Desenho inicial e primeira publicação de posição
	c.desenhar()
	c.publicarPosPlayer()

	tickerRender := time.NewTicker(120 * time.Millisecond) // refresh suave
	defer tickerRender.Stop()

	for {
		select {
		case cmd := <-c.chCmd:
			switch m := cmd.(type) {

			case CmdQuit:
				if c.done != nil {
					close(c.done)
				}
				return

			// --- Teclado como comandos ---
			case CmdMovePlayer:
				// Reutiliza sua lógica de movimentação (mantendo exclusão no coordenador)
				personagemMover(m.Tecla, c.jogo)
				c.publicarPosPlayer()

			case CmdInteragir:
				c.tratarInteragir()

			// --- Infra de inscrição / UI ---
			case CmdStatus:
				c.jogo.StatusMsg = m.Texto

			case CmdSubscribePlayerPos:
				c.subsPos = append(c.subsPos, m.Ch)

			case CmdRegisterInteractable:
				c.interactables[[2]int{m.X, m.Y}] = m.Ch

			// --- Mutação de mapa/entidades (somente aqui) ---
			case CmdSetCell:
				if jogoDentro(c.jogo, m.X, m.Y) {
					jogoSetCelula(c.jogo, m.X, m.Y, m.Elem)
				}

			case CmdTeleportPlayer:
				if jogoPodeMoverPara(c.jogo, m.X, m.Y) {
					// Teleporta o jogador (respeitando último visitado)
					jogoSetCelula(c.jogo, c.jogo.PosX, c.jogo.PosY, c.jogo.UltimoVisitado)
					c.jogo.UltimoVisitado = jogoCelula(c.jogo, m.X, m.Y)
					c.jogo.PosX, c.jogo.PosY = m.X, m.Y
					c.publicarPosPlayer()
				}

			case CmdTryMoveEntity:
				// Valida destino e colisão simples
				if !jogoDentro(c.jogo, m.To.X, m.To.Y) {
					break
				}
				dest := jogoCelula(c.jogo, m.To.X, m.To.Y)
				if dest.tangivel && !m.CanOverlap {
					break
				}
				// Atualiza mapa (apaga origem, desenha destino)
				jogoSetCelula(c.jogo, m.From.X, m.From.Y, Vazio)
				jogoSetCelula(c.jogo, m.To.X, m.To.Y, m.Elem)
			}

			c.desenhar()

		case <-tickerRender.C:
			// Re-render periódico para efeitos de UI
			c.desenhar()
		}
	}
}

func (c *coordenador) tratarInteragir() {
	x, y := c.jogo.PosX, c.jogo.PosY
	deltas := [][2]int{{0, 0}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}} // célula e vizinhos 4-neigh
	for _, d := range deltas {
		if ch, ok := c.interactables[[2]int{x + d[0], y + d[1]}]; ok {
			select {
			case ch <- struct{}{}:
			default:
			}
			return
		}
	}
	c.jogo.StatusMsg = "Nada para interagir por perto."
}

// === Teclado → Comandos (goroutine separada) ===
// Lê eventos crus da interface e os traduz para comandos enviados ao coordenador.
func capturarTeclado(chCmd chan<- Cmd) {
	for {
		ev := interfaceLerEventoTeclado()
		switch ev.Tipo {
		case "sair":
			chCmd <- CmdQuit{}
			return
		case "interagir":
			chCmd <- CmdInteragir{}
		case "mover":
			chCmd <- CmdMovePlayer{Tecla: ev.Tecla} // ev.Tecla já é rune
		}
	}
}
