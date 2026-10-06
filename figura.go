package main

import (
	"fmt"
	"html"
	"html/template"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// RenderFigura desenha a figura de uma questão nova como SVG inline (ou tabela HTML).
// Nada de biblioteca no navegador: o servidor manda a figura pronta.
func RenderFigura(f *Figura) template.HTML {
	if f == nil {
		return ""
	}
	switch f.Lib {
	case "jsxgraph", "geo", "geometria":
		return template.HTML(svgGeo(f))
	case "chart", "grafico":
		return template.HTML(svgChart(f))
	case "tabela":
		return template.HTML(tabelaHTML(f))
	}
	return ""
}

func esc(s string) string { return html.EscapeString(s) }

func num(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) }

// ---------- geometria e funções ----------

func svgGeo(f *Figura) string {
	bb := f.BBox
	if len(bb) != 4 {
		bb = []float64{-5, 5, 5, -5}
	}
	xmin, ymax, xmax, ymin := bb[0], bb[1], bb[2], bb[3]
	if xmax <= xmin || ymax <= ymin {
		return ""
	}
	// Geometria pura: mesma escala em x e y (círculo redondo, ângulo certo); "altura" só limita a exibição.
	// Gráfico com eixos ou função: cada eixo com a sua escala, numa caixa de altura fixa
	// (senão y de 0 a 1400 com x de 0 a 20 vira uma tira altíssima).
	W := 420.0
	grafico := f.Eixos
	for _, e := range f.Elementos {
		if e.T == "funcao" {
			grafico = true
		}
	}
	H := W * (ymax - ymin) / (xmax - xmin)
	estilo := ""
	if grafico {
		H = 280
		if f.Altura > 0 {
			H = math.Min(float64(f.Altura), 420)
		}
	} else if f.Altura > 0 {
		estilo = fmt.Sprintf(` style="max-height:%dpx"`, f.Altura)
	}
	X := func(x float64) float64 { return (x - xmin) / (xmax - xmin) * W }
	Y := func(y float64) float64 { return (ymax - y) / (ymax - ymin) * H }
	sx, sy := W/(xmax-xmin), H/(ymax-ymin)

	var b strings.Builder
	fmt.Fprintf(&b, `<svg class="fig" viewBox="-14 -14 %s %s"%s role="img" aria-label="Figura da questão">`, num(W+28), num(H+28), estilo)

	if f.Eixos {
		if 0 >= ymin && 0 <= ymax {
			fmt.Fprintf(&b, `<line class="ax" x1="0" y1="%s" x2="%s" y2="%s"/>`, num(Y(0)), num(W), num(Y(0)))
		}
		if 0 >= xmin && 0 <= xmax {
			fmt.Fprintf(&b, `<line class="ax" x1="%s" y1="0" x2="%s" y2="%s"/>`, num(X(0)), num(X(0)), num(H))
		}
		passo := tick(xmax - xmin)
		for x := math.Ceil(xmin/passo) * passo; x <= xmax; x += passo {
			if math.Abs(x) < 1e-9 {
				continue
			}
			fmt.Fprintf(&b, `<line class="tk" x1="%s" y1="%s" x2="%s" y2="%s"/><text class="tl" x="%s" y="%s" text-anchor="middle">%s</text>`,
				num(X(x)), num(Y(0)-3), num(X(x)), num(Y(0)+3), num(X(x)), num(Y(0)+14), rotNum(x))
		}
		passo = tick(ymax - ymin)
		for y := math.Ceil(ymin/passo) * passo; y <= ymax; y += passo {
			if math.Abs(y) < 1e-9 {
				continue
			}
			fmt.Fprintf(&b, `<line class="tk" x1="%s" y1="%s" x2="%s" y2="%s"/><text class="tl" x="%s" y="%s" text-anchor="end">%s</text>`,
				num(X(0)-3), num(Y(y)), num(X(0)+3), num(Y(y)), num(X(0)-6), num(Y(y)+4), rotNum(y))
		}
	}

	pts := map[string][2]float64{}
	for _, e := range f.Elementos {
		if e.T == "ponto" && len(e.XY) == 2 {
			pts[e.ID] = [2]float64{e.XY[0], e.XY[1]}
		}
	}
	// desenha preenchimentos primeiro, pontos e textos por último
	for _, fase := range []int{0, 1, 2} {
		for _, e := range f.Elementos {
			switch {
			case fase == 0 && e.T == "poligono":
				var ps []string
				for _, id := range e.Pts {
					if p, ok := pts[id]; ok {
						ps = append(ps, num(X(p[0]))+","+num(Y(p[1])))
					}
				}
				fmt.Fprintf(&b, `<polygon class="pg" points="%s"/>`, strings.Join(ps, " "))
			case fase == 1 && e.T == "segmento":
				p, ok1 := pts[e.De]
				q, ok2 := pts[e.Ate]
				if !ok1 || !ok2 {
					continue
				}
				fmt.Fprintf(&b, `<line class="ln" x1="%s" y1="%s" x2="%s" y2="%s"/>`, num(X(p[0])), num(Y(p[1])), num(X(q[0])), num(Y(q[1])))
				if e.Rotulo != "" {
					mx, my := (X(p[0])+X(q[0]))/2, (Y(p[1])+Y(q[1]))/2
					dx, dy := X(q[0])-X(p[0]), Y(q[1])-Y(p[1])
					l := math.Hypot(dx, dy)
					if l > 0 {
						mx, my = mx+dy/l*12, my-dx/l*12
					}
					fmt.Fprintf(&b, `<text class="lb" x="%s" y="%s" text-anchor="middle">%s</text>`, num(mx), num(my+4), esc(e.Rotulo))
				}
			case fase == 1 && e.T == "circulo":
				c, ok := pts[e.Centro]
				if !ok && len(e.XY) == 2 {
					c, ok = [2]float64{e.XY[0], e.XY[1]}, true
				}
				if ok {
					fmt.Fprintf(&b, `<ellipse class="ln nf" cx="%s" cy="%s" rx="%s" ry="%s"/>`, num(X(c[0])), num(Y(c[1])), num(e.Raio*sx), num(e.Raio*sy))
				}
			case fase == 1 && e.T == "funcao":
				expr, err := ParseExpr(e.Expr)
				if err != nil {
					continue
				}
				a, z := xmin, xmax
				if len(e.Dominio) == 2 {
					a, z = e.Dominio[0], e.Dominio[1]
				}
				var d strings.Builder
				pen := false
				for i := 0; i <= 300; i++ {
					x := a + (z-a)*float64(i)/300
					y := expr(x)
					if math.IsNaN(y) || math.IsInf(y, 0) || y < ymin-(ymax-ymin) || y > ymax+(ymax-ymin) {
						pen = false
						continue
					}
					if pen {
						d.WriteString(" L")
					} else {
						d.WriteString(" M")
						pen = true
					}
					d.WriteString(num(X(x)) + " " + num(Y(y)))
				}
				fmt.Fprintf(&b, `<path class="fn" d="%s"/>`, d.String())
				if e.Rotulo != "" {
					y := expr(z)
					fmt.Fprintf(&b, `<text class="lb" x="%s" y="%s" text-anchor="end">%s</text>`, num(X(z)-4), num(Y(y)-8), esc(e.Rotulo))
				}
			case fase == 1 && e.T == "angulo" && len(e.Pts) == 3:
				p, ok1 := pts[e.Pts[0]]
				v, ok2 := pts[e.Pts[1]]
				q, ok3 := pts[e.Pts[2]]
				if !ok1 || !ok2 || !ok3 {
					continue
				}
				a1 := math.Atan2(-(Y(p[1]) - Y(v[1])), X(p[0])-X(v[0]))
				a2 := math.Atan2(-(Y(q[1]) - Y(v[1])), X(q[0])-X(v[0]))
				dA := a2 - a1
				for dA < 0 {
					dA += 2 * math.Pi
				}
				if dA > math.Pi { // usa sempre o ângulo interno
					a1, a2 = a2, a1
					dA = 2*math.Pi - dA
				}
				r := 18.0
				x1, y1 := X(v[0])+r*math.Cos(a1), Y(v[1])-r*math.Sin(a1)
				x2, y2 := X(v[0])+r*math.Cos(a1+dA), Y(v[1])-r*math.Sin(a1+dA)
				fmt.Fprintf(&b, `<path class="ln nf" d="M%s %s A%s %s 0 0 0 %s %s"/>`, num(x1), num(y1), num(r), num(r), num(x2), num(y2))
				if e.Rotulo != "" {
					am := a1 + dA/2
					fmt.Fprintf(&b, `<text class="lb" x="%s" y="%s" text-anchor="middle">%s</text>`, num(X(v[0])+(r+12)*math.Cos(am)), num(Y(v[1])-(r+12)*math.Sin(am)+4), esc(e.Rotulo))
				}
			case fase == 2 && e.T == "ponto" && len(e.XY) == 2:
				fmt.Fprintf(&b, `<circle class="pt" cx="%s" cy="%s" r="3.2"/>`, num(X(e.XY[0])), num(Y(e.XY[1])))
				if e.Rotulo != "" {
					fmt.Fprintf(&b, `<text class="lb" x="%s" y="%s">%s</text>`, num(X(e.XY[0])+6), num(Y(e.XY[1])-6), esc(e.Rotulo))
				}
			case fase == 2 && e.T == "texto" && len(e.XY) == 2:
				fmt.Fprintf(&b, `<text class="lb" x="%s" y="%s" text-anchor="middle">%s</text>`, num(X(e.XY[0])), num(Y(e.XY[1])), esc(e.Txt))
			}
		}
	}
	b.WriteString(`</svg>`)
	return b.String()
}

func tick(r float64) float64 {
	for _, p := range []float64{1, 2, 5, 10, 20, 50, 100, 200, 500, 1000} {
		if r/p <= 12 {
			return p
		}
	}
	return math.Pow(10, math.Ceil(math.Log10(r/10)))
}

func rotNum(v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	return strings.Replace(strings.Replace(s, ".", ",", 1), "-", "−", 1)
}

// ---------- gráficos de dados ----------

func svgChart(f *Figura) string {
	if len(f.Series) == 0 {
		return ""
	}
	var b strings.Builder
	if f.Tipo == "pie" || f.Tipo == "pizza" {
		dados := f.Series[0].Dados
		tot := 0.0
		for _, v := range dados {
			tot += v
		}
		if tot <= 0 {
			return ""
		}
		fmt.Fprintf(&b, `<figure class="figc"><svg class="fig" viewBox="0 0 420 240" role="img" aria-label="%s">`, esc(f.Titulo))
		cx, cy, r := 120.0, 120.0, 100.0
		a := -math.Pi / 2
		for i, v := range dados {
			da := v / tot * 2 * math.Pi
			x1, y1 := cx+r*math.Cos(a), cy+r*math.Sin(a)
			x2, y2 := cx+r*math.Cos(a+da), cy+r*math.Sin(a+da)
			large := 0
			if da > math.Pi {
				large = 1
			}
			fmt.Fprintf(&b, `<path class="s%d" d="M%s %s L%s %s A%s %s 0 %d 1 %s %s Z"/>`, i%6, num(cx), num(cy), num(x1), num(y1), num(r), num(r), large, num(x2), num(y2))
			rot := ""
			if i < len(f.Rotulos) {
				rot = f.Rotulos[i]
			}
			fmt.Fprintf(&b, `<rect class="s%d" x="250" y="%d" width="12" height="12" rx="2"/><text class="tl" x="268" y="%d">%s (%s%%)</text>`, i%6, 30+i*22, 40+i*22, esc(rot), rotNum(math.Round(v/tot*1000)/10))
			a += da
		}
		b.WriteString(`</svg>`)
		if f.Titulo != "" {
			fmt.Fprintf(&b, `<figcaption>%s</figcaption>`, esc(f.Titulo))
		}
		b.WriteString(`</figure>`)
		return b.String()
	}

	n := len(f.Rotulos)
	for _, s := range f.Series {
		if len(s.Dados) > n {
			n = len(s.Dados)
		}
	}
	maxV := 0.0
	for _, s := range f.Series {
		for _, v := range s.Dados {
			maxV = math.Max(maxV, v)
		}
	}
	if maxV <= 0 || n == 0 {
		return ""
	}
	passo := tick(maxV)
	topo := math.Ceil(maxV/passo) * passo
	W, H, L, R, T, B := 460.0, 260.0, 48.0, 12.0, 14.0, 40.0
	X0, Y0 := L, H-B
	pw, ph := W-L-R, H-T-B
	Y := func(v float64) float64 { return Y0 - v/topo*ph }
	fmt.Fprintf(&b, `<figure class="figc"><svg class="fig" viewBox="0 0 %s %s" role="img" aria-label="%s">`, num(W), num(H), esc(f.Titulo))
	for v := 0.0; v <= topo+1e-9; v += passo {
		fmt.Fprintf(&b, `<line class="gr" x1="%s" x2="%s" y1="%s" y2="%s"/><text class="tl" x="%s" y="%s" text-anchor="end">%s</text>`, num(X0), num(W-R), num(Y(v)), num(Y(v)), num(X0-6), num(Y(v)+4), rotNum(v))
	}
	slot := pw / float64(n)
	for i := 0; i < n; i++ {
		rot := ""
		if i < len(f.Rotulos) {
			rot = f.Rotulos[i]
		}
		fmt.Fprintf(&b, `<text class="tl" x="%s" y="%s" text-anchor="middle">%s</text>`, num(X0+slot*(float64(i)+.5)), num(Y0+16), esc(rot))
	}
	if f.Tipo == "line" || f.Tipo == "linha" {
		for si, s := range f.Series {
			var d strings.Builder
			for i, v := range s.Dados {
				x := X0 + slot*(float64(i)+.5)
				if i == 0 {
					d.WriteString("M")
				} else {
					d.WriteString(" L")
				}
				d.WriteString(num(x) + " " + num(Y(v)))
				fmt.Fprintf(&b, `<circle class="s%d" cx="%s" cy="%s" r="3.5"/>`, si%6, num(x), num(Y(v)))
			}
			fmt.Fprintf(&b, `<path class="lns s%d" d="%s"/>`, si%6, d.String())
		}
	} else {
		ns := float64(len(f.Series))
		bw := slot * 0.7 / ns
		for si, s := range f.Series {
			for i, v := range s.Dados {
				x := X0 + slot*float64(i) + slot*0.15 + bw*float64(si)
				fmt.Fprintf(&b, `<rect class="s%d" x="%s" y="%s" width="%s" height="%s" rx="2"/>`, si%6, num(x), num(Y(v)), num(bw-2), num(Y0-Y(v)))
			}
		}
	}
	fmt.Fprintf(&b, `<line class="ax" x1="%s" x2="%s" y1="%s" y2="%s"/>`, num(X0), num(W-R), num(Y0), num(Y0))
	if f.EixoY != "" {
		fmt.Fprintf(&b, `<text class="tl" x="4" y="10">%s</text>`, esc(f.EixoY))
	}
	b.WriteString(`</svg>`)
	if len(f.Series) > 1 {
		b.WriteString(`<div class="leg">`)
		for si, s := range f.Series {
			fmt.Fprintf(&b, `<span><i class="s%d"></i>%s</span>`, si%6, esc(s.Nome))
		}
		b.WriteString(`</div>`)
	}
	if f.Titulo != "" {
		fmt.Fprintf(&b, `<figcaption>%s</figcaption>`, esc(f.Titulo))
	}
	b.WriteString(`</figure>`)
	return b.String()
}

func tabelaHTML(f *Figura) string {
	var b strings.Builder
	b.WriteString(`<div class="tbl"><table><thead><tr>`)
	for _, c := range f.Cabecalho {
		fmt.Fprintf(&b, `<th>%s</th>`, esc(c))
	}
	b.WriteString(`</tr></thead><tbody>`)
	for _, l := range f.Linhas {
		b.WriteString(`<tr>`)
		for _, c := range l {
			fmt.Fprintf(&b, `<td>%s</td>`, esc(c))
		}
		b.WriteString(`</tr>`)
	}
	b.WriteString(`</tbody></table></div>`)
	return b.String()
}

// ---------- expressões: x^2 - 2*x, sqrt(x), 2x, sen(x)… ----------

type fnx func(float64) float64

type parser struct {
	s   []rune
	pos int
}

func ParseExpr(s string) (fnx, error) {
	p := &parser{s: []rune(strings.ReplaceAll(s, "**", "^"))}
	f, err := p.expr()
	if err != nil {
		return nil, err
	}
	p.ws()
	if p.pos < len(p.s) {
		return nil, fmt.Errorf("sobra em %q", string(p.s[p.pos:]))
	}
	return f, nil
}

func (p *parser) ws() {
	for p.pos < len(p.s) && unicode.IsSpace(p.s[p.pos]) {
		p.pos++
	}
}
func (p *parser) peek() rune {
	p.ws()
	if p.pos < len(p.s) {
		return p.s[p.pos]
	}
	return 0
}

func (p *parser) expr() (fnx, error) {
	l, err := p.term()
	if err != nil {
		return nil, err
	}
	for {
		switch p.peek() {
		case '+', '-':
			op := p.s[p.pos]
			p.pos++
			r, err := p.term()
			if err != nil {
				return nil, err
			}
			a, b := l, r
			if op == '+' {
				l = func(x float64) float64 { return a(x) + b(x) }
			} else {
				l = func(x float64) float64 { return a(x) - b(x) }
			}
		default:
			return l, nil
		}
	}
}

func (p *parser) term() (fnx, error) {
	l, err := p.unary()
	if err != nil {
		return nil, err
	}
	for {
		c := p.peek()
		if c == '*' || c == '/' {
			p.pos++
			r, err := p.unary()
			if err != nil {
				return nil, err
			}
			a, b := l, r
			if c == '*' {
				l = func(x float64) float64 { return a(x) * b(x) }
			} else {
				l = func(x float64) float64 { return a(x) / b(x) }
			}
			continue
		}
		// multiplicação implícita: 2x, 3(x+1), x sqrt(x)
		if c == '(' || unicode.IsLetter(c) {
			r, err := p.unary()
			if err != nil {
				return nil, err
			}
			a, b := l, r
			l = func(x float64) float64 { return a(x) * b(x) }
			continue
		}
		return l, nil
	}
}

func (p *parser) unary() (fnx, error) {
	if p.peek() == '-' {
		p.pos++
		u, err := p.unary()
		if err != nil {
			return nil, err
		}
		return func(x float64) float64 { return -u(x) }, nil
	}
	if p.peek() == '+' {
		p.pos++
		return p.unary()
	}
	return p.power()
}

func (p *parser) power() (fnx, error) {
	base, err := p.primary()
	if err != nil {
		return nil, err
	}
	if p.peek() == '^' {
		p.pos++
		ex, err := p.unary()
		if err != nil {
			return nil, err
		}
		return func(x float64) float64 { return math.Pow(base(x), ex(x)) }, nil
	}
	return base, nil
}

var funcs = map[string]func(float64) float64{
	"sqrt": math.Sqrt, "raiz": math.Sqrt, "abs": math.Abs, "sin": math.Sin, "sen": math.Sin, "cos": math.Cos,
	"tan": math.Tan, "tg": math.Tan, "log": math.Log10, "ln": math.Log, "exp": math.Exp,
}

func (p *parser) primary() (fnx, error) {
	c := p.peek()
	switch {
	case c == '(':
		p.pos++
		e, err := p.expr()
		if err != nil {
			return nil, err
		}
		if p.peek() != ')' {
			return nil, fmt.Errorf("falta )")
		}
		p.pos++
		return e, nil
	case unicode.IsDigit(c) || c == '.' || c == ',':
		st := p.pos
		for p.pos < len(p.s) && (unicode.IsDigit(p.s[p.pos]) || p.s[p.pos] == '.' || p.s[p.pos] == ',') {
			p.pos++
		}
		v, err := strconv.ParseFloat(strings.Replace(string(p.s[st:p.pos]), ",", ".", 1), 64)
		if err != nil {
			return nil, err
		}
		return func(float64) float64 { return v }, nil
	case unicode.IsLetter(c):
		st := p.pos
		for p.pos < len(p.s) && unicode.IsLetter(p.s[p.pos]) {
			p.pos++
		}
		id := strings.ToLower(string(p.s[st:p.pos]))
		switch id {
		case "x":
			return func(x float64) float64 { return x }, nil
		case "pi", "π":
			return func(float64) float64 { return math.Pi }, nil
		case "e":
			return func(float64) float64 { return math.E }, nil
		}
		if f, ok := funcs[id]; ok {
			arg, err := p.primary()
			if err != nil {
				return nil, err
			}
			return func(x float64) float64 { return f(arg(x)) }, nil
		}
		return nil, fmt.Errorf("nome desconhecido %q", id)
	}
	return nil, fmt.Errorf("expressão inválida")
}
