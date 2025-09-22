package main

import (
	"math/rand"
	"time"
)

// ---------- Alavanca ----------

func iniciarAlavanca(x, y int, chCmd chan<- Cmd, outAbrir chan<- sinal) {
	chAtivar := make(chan struct{}, 1)
	// registra como “interagível” no coordenador
	chCmd <- CmdRegistrarInteragivel{X: x, Y: y, Ch: chAtivar}
	// desenha desligada
	chCmd <- CmdSetCelula{X: x, Y: y, Elem: AlavancaOff}

	go func() {
		ligada := false
		for range chAtivar {
			ligada = !ligada
			if ligada {
				chCmd <- CmdSetCelula{X: x, Y: y, Elem: AlavancaOn}
				chCmd <- CmdStatus{Texto: "Alavanca ativada!"}
				// dispara abertura do portal (sem bloquear)
				select {
				case outAbrir <- sinal{}:
				default:
				}
			} else {
				chCmd <- CmdSetCelula{X: x, Y: y, Elem: AlavancaOff}
				chCmd <- CmdStatus{Texto: "Alavanca desativada!"}
			}
		}
	}()
}

// ---------- Portal (com timeout) ----------

type sinal struct{}

func iniciarPortal(x, y int, destino Ponto, chCmd chan<- Cmd, chAbrir <-chan sinal, chPosPlayer <-chan Ponto) {
	go func() {
		aberto := false
		fecha := func() { aberto = false; chCmd <- CmdSetCelula{X: x, Y: y, Elem: PortalFechado} }
		abre := func() { aberto = true; chCmd <- CmdSetCelula{X: x, Y: y, Elem: PortalAberto} }

		fecha() // começa fechado
		for {
			select {
			case <-chAbrir:
				abre()
				chCmd <- CmdStatus{Texto: "Portal aberto! (5s)"}
				// timeout: fecha se não utilizado
				select {
				case <-time.After(5 * time.Second):
					fecha()
					chCmd <- CmdStatus{Texto: "Portal fechou por timeout."}
				case p := <-chPosPlayer:
					if aberto && p.X == x && p.Y == y {
						chCmd <- CmdTeleportarJogador{X: destino.X, Y: destino.Y}
						fecha()
						chCmd <- CmdStatus{Texto: "Teleportado pelo portal!"}
					}
				}

			case p := <-chPosPlayer:
				// jogador entrou depois de aberto (sem aguardar novo abrir)
				if aberto && p.X == x && p.Y == y {
					chCmd <- CmdTeleportarJogador{X: destino.X, Y: destino.Y}
					fecha()
					chCmd <- CmdStatus{Texto: "Teleportado pelo portal!"}
				}
			}
		}
	}()
}

// ---------- Sentinela (escuta múltiplos canais) ----------

func iniciarSentinela(id string, start Ponto, waypoints []Ponto, raioPerseguir int,
	chCmd chan<- Cmd, chPosPlayer <-chan Ponto) {

	go func() {
		pos := start
		chCmd <- CmdSetCelula{X: pos.X, Y: pos.Y, Elem: SentinelaElem}

		i := 0
		modoPerseguir := false
		ultimoVisto := start
		tick := time.NewTicker(350 * time.Millisecond)
		defer tick.Stop()

		for {
			select {
			case <-tick.C:
				// movimento periódico
				alvo := waypoints[i]
				if modoPerseguir {
					alvo = ultimoVisto
				}
				prox := passoRumo(pos, alvo)
				if prox != pos {
					chCmd <- CmdTryMoveElemento{
						ID: id, From: pos, To: prox, Elem: SentinelaElem,
					}
					pos = prox
				}
				if pos == alvo && !modoPerseguir {
					i = (i + 1) % len(waypoints)
				}

			case p := <-chPosPlayer:
				// escuta outro canal simultaneamente: alterna comportamento
				ultimoVisto = p
				if distManhattan(pos, p) <= raioPerseguir {
					modoPerseguir = true
				} else if modoPerseguir && distManhattan(pos, p) > (raioPerseguir+2) {
					modoPerseguir = false // perdeu o jogador
				}
			}
		}
	}()
}

func passoRumo(atual, alvo Ponto) Ponto {
	dx, dy := 0, 0
	if alvo.X > atual.X {
		dx = 1
	} else if alvo.X < atual.X {
		dx = -1
	}
	if alvo.Y > atual.Y {
		dy = 1
	} else if alvo.Y < atual.Y {
		dy = -1
	}
	// opcional: prioriza eixo com maior delta
	if abs(alvo.X-atual.X) >= abs(alvo.Y-atual.Y) {
		return Ponto{atual.X + dx, atual.Y}
	}
	return Ponto{atual.X, atual.Y + dy}
}

func distManhattan(a, b Ponto) int { return abs(a.X-b.X) + abs(a.Y-b.Y) }
func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// ---------- Armadilha oscilante (extra visível) ----------

func iniciarArmadilha(x, y int, onMs, offMs int, chCmd chan<- Cmd, chPosPlayer <-chan Ponto) {
	go func() {
		ativa := false
		set := func() {
			if ativa {
				chCmd <- CmdSetCelula{X: x, Y: y, Elem: ArmadilhaOn}
			} else if !ativa {
				chCmd <- CmdSetCelula{X: x, Y: y, Elem: ArmadilhaOff}
			}
		}
		set()

		tOn := time.NewTicker(time.Duration(onMs) * time.Millisecond)
		tOff := time.NewTicker(time.Duration(offMs) * time.Millisecond)
		defer tOn.Stop()
		defer tOff.Stop()

		for {
			select {
			case <-tOn.C:
				ativa = true
				set()
			case <-tOff.C:
				ativa = false
				set()
			case p := <-chPosPlayer:
				if ativa && p.X == x && p.Y == y {
					chCmd <- CmdStatus{Texto: "Aouch! Dano da armadilha."}
				}
			}
		}
	}()
}

// ---------- util pequeno para escolher células vazias (opcional) ----------
func escolherVazioLargo(j *Jogo) Ponto {
	rand.Seed(time.Now().UnixNano())
	for {
		y := rand.Intn(len(j.Mapa))
		if len(j.Mapa[y]) == 0 {
			continue
		}
		x := rand.Intn(len(j.Mapa[y]))
		if !j.Mapa[y][x].tangivel {
			return Ponto{X: x, Y: y}
		}
	}
}
