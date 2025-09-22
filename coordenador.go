package main

import (
	"time"
)

type coordenador struct {
	jogo          *Jogo
	chTeclado     <-chan EventoTeclado
	chCmd         <-chan Cmd
	subsPos       []chan<- Ponto
	interactables map[[2]int]chan<- struct{}
}

func novoCoordenador(j *Jogo, chTeclado <-chan EventoTeclado, chCmd <-chan Cmd) *coordenador {
	return &coordenador{
		jogo:          j,
		chTeclado:     chTeclado,
		chCmd:         chCmd,
		interactables: make(map[[2]int]chan<- struct{}),
	}
}

func (c *coordenador) publicarPosPlayer() {
	p := Ponto{c.jogo.PosX, c.jogo.PosY}
	for _, ch := range c.subsPos {
		select { // não bloquear o coordenador
		case ch <- p:
		default:
		}
	}
}

func (c *coordenador) desenhar() { interfaceDesenharJogo(c.jogo) }

func (c *coordenador) loop() {
	c.desenhar()
	tickerRender := time.NewTicker(120 * time.Millisecond) // refresh suave
	defer tickerRender.Stop()

	for {
		select {
		case ev := <-c.chTeclado:
			switch ev.Tipo {
			case "sair":
				return
			case "interagir":
				c.tratarInteragir()
			case "mover":
				// reutiliza sua lógica existente para mover o jogador
				personagemMover(ev.Tecla, c.jogo)
				c.publicarPosPlayer()
			}
			c.desenhar()

		case cmd := <-c.chCmd:
			switch m := cmd.(type) {
			case CmdQuit:
				return

			case CmdStatus:
				c.jogo.StatusMsg = m.Texto

			case CmdEscutarPosDoJogador:
				c.subsPos = append(c.subsPos, m.Ch)

			case CmdRegistrarInteragivel:
				c.interactables[[2]int{m.X, m.Y}] = m.Ch

			case CmdSetCelula:
				// exclusão mútua: apenas o coordenador altera o mapa
				if jogoDentro(c.jogo, m.X, m.Y) {
					jogoSetCelula(c.jogo, m.X, m.Y, m.Elem)
				}

			case CmdTeleportarJogador:
				if jogoPodeMoverPara(c.jogo, m.X, m.Y) {
					// move “teleportando”: restaura onde está e grava no destino
					jogoSetCelula(c.jogo, c.jogo.PosX, c.jogo.PosY, c.jogo.UltimoVisitado)
					c.jogo.UltimoVisitado = jogoCelula(c.jogo, m.X, m.Y)
					c.jogo.PosX, c.jogo.PosY = m.X, m.Y
					c.publicarPosPlayer()
				}

			case CmdTryMoveElemento:
				// valida colisões simples com o mapa
				if !jogoDentro(c.jogo, m.To.X, m.To.Y) {
					break
				}
				dest := jogoCelula(c.jogo, m.To.X, m.To.Y)
				if dest.tangivel && !m.PodeSobrepor {
					break
				}
				// apaga posição antiga e desenha nova (visual direto no mapa)
				jogoSetCelula(c.jogo, m.From.X, m.From.Y, Vazio)
				jogoSetCelula(c.jogo, m.To.X, m.To.Y, m.Elem)
			}
			c.desenhar()

		case <-tickerRender.C:
			// redesenha periodicamente (efeitos de status etc.)
			c.desenhar()
		}
	}
}

func (c *coordenador) tratarInteragir() {
	x, y := c.jogo.PosX, c.jogo.PosY
	deltas := [][2]int{{0, 0}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}} // célula e adjacências 4-neigh
	for _, d := range deltas {
		ch, ok := c.interactables[[2]int{x + d[0], y + d[1]}]
		if ok {
			select {
			case ch <- struct{}{}:
			default:
			}
			return
		}
	}
	c.jogo.StatusMsg = "Nada para interagir por perto."
}

// **Captura de teclado em goroutine separada**
func capturarTeclado(ch chan<- EventoTeclado) {
	for {
		ev := interfaceLerEventoTeclado()
		ch <- ev
		if ev.Tipo == "sair" {
			return
		}
	}
}
