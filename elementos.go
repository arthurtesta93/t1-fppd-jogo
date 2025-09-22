package main

import (
	"time"
)

// ---------- Alavanca ----------

type sinal struct{}

func iniciarAlavanca(x, y int, chCmd chan<- Cmd, outAbrir chan<- sinal) {
	chAtivar := make(chan struct{}, 1)
	chCmd <- CmdRegisterInteractable{X: x, Y: y, Ch: chAtivar}
	chCmd <- CmdSetCell{X: x, Y: y, Elem: AlavancaOff}

	go func() {
		ligada := false
		for range chAtivar {
			ligada = !ligada
			if ligada {
				chCmd <- CmdSetCell{X: x, Y: y, Elem: AlavancaOn}
				chCmd <- CmdStatus{Texto: "Alavanca ativada!"}
				select {
				case outAbrir <- sinal{}:
				default:
				}
			} else {
				chCmd <- CmdSetCell{X: x, Y: y, Elem: AlavancaOff}
				chCmd <- CmdStatus{Texto: "Alavanca desativada!"}
			}
		}
	}()
}

// ---------- Portal (com timeout) ----------
func iniciarPortal(x, y int, destino Ponto, chCmd chan<- Cmd, chAbrir <-chan sinal, chPosPlayer <-chan Ponto) {
	go func() {
		aberto := false
		fecha := func() { aberto = false; chCmd <- CmdSetCell{X: x, Y: y, Elem: PortalFechado} }
		abre := func() { aberto = true; chCmd <- CmdSetCell{X: x, Y: y, Elem: PortalAberto} }

		fecha() // começa fechado
		for {
			select {
			case <-chAbrir:
				abre()
				chCmd <- CmdStatus{Texto: "Portal aberto! (5s)"}
				select {
				case <-time.After(5 * time.Second):
					fecha()
					chCmd <- CmdStatus{Texto: "Portal fechou por timeout."}
				case p := <-chPosPlayer:
					if aberto && p.X == x && p.Y == y {
						chCmd <- CmdTeleportPlayer{X: destino.X, Y: destino.Y}
						fecha()
						chCmd <- CmdStatus{Texto: "Teleportado pelo portal!"}
					}
				}
			case p := <-chPosPlayer:
				if aberto && p.X == x && p.Y == y {
					chCmd <- CmdTeleportPlayer{X: destino.X, Y: destino.Y}
					fecha()
					chCmd <- CmdStatus{Texto: "Teleportado pelo portal!"}
				}
			}
		}
	}()
}

// ---------- Sentinela e Armadilha (inalterados) ----------
func iniciarSentinela(id string, start Ponto, waypoints []Ponto, raioPerseguir int,
	chCmd chan<- Cmd, chPosPlayer <-chan Ponto) {

	go func() {
		pos := start
		chCmd <- CmdSetCell{X: pos.X, Y: pos.Y, Elem: SentinelaElem}

		i := 0
		modoPerseguir := false
		ultimoVisto := start
		tick := time.NewTicker(350 * time.Millisecond)
		defer tick.Stop()

		for {
			select {
			case <-tick.C:
				alvo := waypoints[i]
				if modoPerseguir {
					alvo = ultimoVisto
				}
				prox := passoRumo(pos, alvo)
				if prox != pos {
					chCmd <- CmdTryMoveEntity{ID: id, From: pos, To: prox, Elem: SentinelaElem}
					pos = prox
				}
				if pos == alvo && !modoPerseguir {
					i = (i + 1) % len(waypoints)
				}
			case p := <-chPosPlayer:
				ultimoVisto = p
				if distManhattan(pos, p) <= raioPerseguir {
					modoPerseguir = true
				} else if modoPerseguir && distManhattan(pos, p) > (raioPerseguir+2) {
					modoPerseguir = false
				}
			}
		}
	}()
}

func iniciarArmadilha(x, y int, onMs, offMs int, chCmd chan<- Cmd, chPosPlayer <-chan Ponto) {
	go func() {
		ativa := false
		set := func() {
			if ativa {
				chCmd <- CmdSetCell{X: x, Y: y, Elem: ArmadilhaOn}
			} else {
				chCmd <- CmdSetCell{X: x, Y: y, Elem: ArmadilhaOff}
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

// --- utils ---
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
