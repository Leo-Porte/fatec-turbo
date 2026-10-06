// Fatec Turbo — todo o progresso fica no localStorage deste navegador (sem login, sem servidor de dados).
(function () {
  "use strict";
  var KEY = "ft27";
  var DISC = ["Português", "Matemática", "Multidisciplinar", "Raciocínio Lógico", "Física", "Química", "Biologia", "História", "Geografia", "Inglês"];

  // ---------- armazenamento ----------
  function vazio() { return { v: 1, rotina: {}, lidas: {}, ex: {}, testes: {}, sims: {}, reds: {} }; }
  function load() {
    try { var d = JSON.parse(localStorage.getItem(KEY) || "null"); if (d && d.v === 1) { var b = vazio(); for (var k in b) if (!d[k]) d[k] = b[k]; return d; } } catch (e) {}
    return vazio();
  }
  var DB = load(), salvoOk = true;
  function save() {
    try { localStorage.setItem(KEY, JSON.stringify(DB)); salvoOk = true; }
    catch (e) { if (salvoOk) toast("Não consegui salvar neste navegador (memória cheia ou modo privado). Exporte seus dados."); salvoOk = false; }
  }
  window.addEventListener("storage", function (e) { if (e.key === KEY) DB = load(); });

  // ---------- utilidades ----------
  function $(s, r) { return (r || document).querySelector(s); }
  function $$(s, r) { return Array.prototype.slice.call((r || document).querySelectorAll(s)); }
  function esc(s) { return String(s == null ? "" : s).replace(/[&<>"']/g, function (c) { return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c]; }); }
  function pct(a, b) { return b ? Math.round(a / b * 100) : 0; }
  function nomeProva(id) { return String(id).replace("-", "/"); }
  function fmtTempo(s) { s = Math.max(0, Math.floor(s)); var h = Math.floor(s / 3600), m = Math.floor(s % 3600 / 60), x = s % 60; return h + ":" + String(m).padStart(2, "0") + ":" + String(x).padStart(2, "0"); }
  function fmtData(ts) { var d = new Date(ts); return d.toLocaleDateString("pt-BR", { day: "2-digit", month: "2-digit" }) + " " + d.toLocaleTimeString("pt-BR", { hour: "2-digit", minute: "2-digit" }); }
  var toastT;
  function toast(msg) { var t = $(".toast"); if (!t) { t = document.createElement("div"); t.className = "toast"; t.setAttribute("role", "status"); document.body.appendChild(t); } t.textContent = msg; t.hidden = false; clearTimeout(toastT); toastT = setTimeout(function () { t.hidden = true; }, 2800); }
  function modal(titulo, corpo, ok, onOk) {
    var m = document.createElement("div"); m.className = "modal";
    m.innerHTML = '<div class="panel" role="dialog" aria-modal="true" aria-labelledby="mt"><h2 id="mt" class="mt0">' + esc(titulo) + '</h2><p>' + esc(corpo) + '</p><div class="row" style="justify-content:flex-end;margin-top:1rem"><button class="btn ghost" type="button" data-x>Voltar</button><button class="btn primary" type="button" data-ok>' + esc(ok) + '</button></div></div>';
    m.querySelector("[data-x]").onclick = function () { m.remove(); };
    m.querySelector("[data-ok]").onclick = function () { m.remove(); onOk(); };
    m.addEventListener("click", function (e) { if (e.target === m) m.remove(); });
    document.body.appendChild(m); m.querySelector("[data-ok]").focus();
  }
  function sims(filtro) {
    return Object.keys(DB.sims).map(function (id) { var s = DB.sims[id]; s.id = id; return s; })
      .filter(function (s) { return s.status === "finalizado" && (!filtro || s.tipo === filtro); })
      .sort(function (a, b) { return a.fim - b.fim; });
  }
  function barra(nome, a, t) { var p = pct(a, t); return '<div class="bar"><span>' + esc(nome) + '</span><i class="' + (p < 50 ? "low" : "") + '"><b style="width:' + p + '%"></b></i><span>' + a + "/" + t + "</span></div>"; }
  function barras(obj, ordem) { return (ordem || Object.keys(obj)).filter(function (k) { return obj[k]; }).map(function (k) { return barra(k, obj[k][0], obj[k][1]); }).join(""); }

  // ---------- comportamentos globais ----------
  document.addEventListener("click", function (e) {
    var z = e.target.closest("[data-zoom]");
    if (z && !z.closest(".zoom")) {
      var o = document.createElement("div"); o.className = "zoom"; o.setAttribute("role", "dialog"); o.setAttribute("aria-label", "Imagem ampliada");
      o.innerHTML = '<button class="btn zoom-close" type="button">Fechar</button><div class="inner">' + z.innerHTML + "</div>";
      o.addEventListener("click", function () { o.remove(); }); document.body.appendChild(o); return;
    }
    var c = e.target.closest("[data-copy]");
    if (c) {
      var alvo = $(c.dataset.copy), txt = alvo ? alvo.value || alvo.textContent : "";
      var ok = function () { toast("Prompt copiado. Cole na IA e anexe o arquivo."); };
      if (navigator.clipboard && navigator.clipboard.writeText) navigator.clipboard.writeText(txt).then(ok, function () { alvo.select(); document.execCommand("copy"); ok(); });
      else { alvo.select(); document.execCommand("copy"); ok(); }
    }
  });
  document.addEventListener("keydown", function (e) { if (e.key === "Escape") { $$(".zoom,.modal").forEach(function (x) { x.remove(); }); } });
  var tb = $("#tema-btn");
  if (tb) tb.onclick = function () {
    var r = document.documentElement, escuro = r.dataset.theme ? r.dataset.theme === "dark" : matchMedia("(prefers-color-scheme: dark)").matches;
    r.dataset.theme = escuro ? "light" : "dark"; try { localStorage.setItem("ft27-tema", r.dataset.theme); } catch (e) {}
  };

  // ---------- INÍCIO ----------
  var rotina = $("#rotina");
  if (rotina) {
    var auto = function (a) {
      if (!a) return false; var p = a.split(":"), t = p[0], v = p.slice(1).join(":");
      if (t === "licao") return !!DB.lidas[v];
      if (t === "teste") return !!(DB.testes[v] && DB.testes[v].corrigido);
      if (t === "sim") return sims("prova").some(function (s) { return s.prova === v; });
      if (t === "red") return !!(DB.reds[v] && DB.reds[v].pdfEm);
      return false;
    };
    var feitas = 0, total = 0;
    $$("li", rotina).forEach(function (li) {
      var c = $("input", li), a = auto(li.dataset.auto); total++;
      if (a) { c.checked = true; c.disabled = true; c.title = "Marcado automaticamente"; }
      else c.checked = !!DB.rotina[c.id];
      if (c.checked) feitas++; li.classList.toggle("done", c.checked);
      c.onchange = function () { if (c.checked) DB.rotina[c.id] = Date.now(); else delete DB.rotina[c.id]; save(); li.classList.toggle("done", c.checked); };
    });
    var ult = sims("prova").pop();
    var st = function (k, v) { var el = $('[data-stat="' + k + '"]'); if (el) el.textContent = v; };
    st("tarefas", feitas + "/" + total); st("ultimo", ult ? ult.resultado.nota : "–");
    st("licoes", Object.keys(DB.lidas).length); st("red", Object.keys(DB.reds).filter(function (k) { return DB.reds[k].pdfEm; }).length);
  }

  // ---------- LIÇÕES (lista) ----------
  $$("[data-licao]").forEach(function (li) { if (DB.lidas[li.dataset.licao]) { var p = $("[data-lida]", li); if (p) p.hidden = false; } });

  // ---------- exercícios (lição: confere na hora; teste: corrige no fim) ----------
  function exercicios(raiz, chave, modoProva) {
    var gab = JSON.parse(raiz.dataset.gab || "[]");
    var store = modoProva ? (DB.testes[chave] = DB.testes[chave] || { resp: {} }).resp : (DB.ex[chave] = DB.ex[chave] || {});
    function pinta(box, revelar) {
      var i = +box.dataset.i, dado = store[i];
      $$(".alts button", box).forEach(function (b) {
        b.classList.remove("sel", "right", "wrong"); b.disabled = false;
        if (revelar) { b.disabled = true; if (b.dataset.l === gab[i]) b.classList.add("right"); else if (b.dataset.l === dado) b.classList.add("wrong"); }
        else if (b.dataset.l === dado) b.classList.add("sel");
      });
      var sol = $("details.sol", box); if (sol) { sol.hidden = !revelar; if (revelar) sol.open = true; }
    }
    var corrigido = modoProva && DB.testes[chave].corrigido;
    $$(".exq", raiz).forEach(function (box) {
      var i = +box.dataset.i;
      pinta(box, modoProva ? corrigido : store[i] != null);
      $$(".alts button", box).forEach(function (b) {
        b.onclick = function () { store[i] = b.dataset.l; save(); pinta(box, !modoProva); };
      });
    });
    var form = $("#rev-form");
    if (form) form.addEventListener("submit", function () { $("#rev-resp").value = JSON.stringify(store); });
    return { store: store, gab: gab, pinta: pinta };
  }
  var licao = $("#licao");
  if (licao) {
    var id = licao.dataset.id;
    if ($("#exs")) exercicios(licao, id, false);
    var bl = $("#marcar-lida");
    var pintaLida = function () { var l = !!DB.lidas[id]; bl.textContent = l ? "Lida ✓ (desmarcar)" : "Marcar como lida"; bl.classList.toggle("primary", !l); };
    bl.onclick = function () { if (DB.lidas[id]) delete DB.lidas[id]; else DB.lidas[id] = Date.now(); save(); pintaLida(); };
    pintaLida();
  }
  var teste = $("#teste");
  if (teste) {
    var tid = teste.dataset.id, ex = exercicios(teste, tid, true);
    var placar = function () { var t = DB.testes[tid]; if (t.corrigido) $("#placar").textContent = "Você acertou " + t.acertos + " de " + t.total + " (" + pct(t.acertos, t.total) + "%)."; };
    $("#corrigir").onclick = function () {
      var falta = ex.gab.length - Object.keys(ex.store).length;
      var faz = function () {
        var t = DB.testes[tid], a = 0; ex.gab.forEach(function (g, i) { if (ex.store[i] === g) a++; });
        t.corrigido = Date.now(); t.acertos = a; t.total = ex.gab.length; save();
        $$(".exq", teste).forEach(function (b) { ex.pinta(b, true); }); placar();
      };
      if (falta > 0) modal("Corrigir agora?", "Faltam " + falta + " questões. Em branco conta como erro.", "Corrigir", faz); else faz();
    };
    placar();
  }
  $$("[data-teste]").forEach(function (li) { var t = DB.testes[li.dataset.teste]; if (t && t.corrigido) { var p = $("[data-feito]", li); p.hidden = false; p.textContent = t.acertos + "/" + t.total; } });

  // ---------- SIMULADOS (lista + treino) ----------
  var ft = $("#form-treino");
  if (ft) {
    var temas = JSON.parse($("#temas-json").textContent || "{}"), sd = $("#tr-disc"), stm = $("#tr-tema"), sn = $("#tr-n");
    var conta = function () {
      var n = 0; Object.keys(temas).forEach(function (d) { if (sd.value && d !== sd.value) return; Object.keys(temas[d]).forEach(function (t) { if (!stm.value || t === stm.value) n += temas[d][t]; }); });
      sn.textContent = n + " questões disponíveis";
    };
    var enche = function () {
      var c = {}; Object.keys(temas).forEach(function (d) { if (sd.value && d !== sd.value) return; Object.keys(temas[d]).forEach(function (t) { c[t] = (c[t] || 0) + temas[d][t]; }); });
      stm.innerHTML = '<option value="">Todos</option>' + Object.keys(c).sort(function (a, b) { return c[b] - c[a]; }).map(function (t) { return '<option value="' + esc(t) + '">' + esc(t) + " (" + c[t] + ")</option>"; }).join("");
      conta();
    };
    sd.onchange = enche; stm.onchange = conta; enche();
    ft.addEventListener("submit", function () {
      var v = {}; sims("treino").forEach(function (s) { (s.itens || []).forEach(function (k) { v[k] = 1; }); });
      $("#tr-visto").value = Object.keys(v).slice(-300).join(",");
    });
    $$("[data-prova]").forEach(function (li) {
      var p = li.dataset.prova, feitos = sims("prova").filter(function (s) { return s.prova === p; }), ult = feitos.pop();
      var and = Object.keys(DB.sims).some(function (k) { var s = DB.sims[k]; return s.tipo === "prova" && s.prova === p && s.status === "andamento"; });
      if (ult) { var n = $("[data-nota]", li); n.hidden = false; n.textContent = "nota " + ult.resultado.nota; }
      if (and) $("[data-andamento]", li).hidden = false;
      $("[data-acao]", li).textContent = and ? "Continuar" : ult ? "Ver / refazer" : "Começar";
    });
    var hist = sims().reverse().slice(0, 15);
    if (hist.length) $("#historico").innerHTML = hist.map(function (s) {
      var href = s.tipo === "prova" ? "/simulado/" + s.prova + "?ver=" + s.id : "/treino?ver=" + s.id;
      return '<li><div class="row"><div class="grow"><b class="disp">' + esc(s.titulo) + '</b> <span class="small muted disp">' + fmtData(s.fim) + '</span></div><span class="pill ' + (s.resultado.nota >= 60 ? "good" : "bad") + '">' + s.resultado.acertos + "/" + s.resultado.total + '</span><a class="btn ghost small" href="' + href + '">Ver</a></div></li>';
    }).join("");
  }

  // ---------- EXECUTOR de simulado / treino ----------
  var exe = $("#executor");
  if (exe) executor();
  function executor() {
    var cfg = JSON.parse($("#cfg").textContent);
    var url = new URL(location.href), ver = url.searchParams.get("ver");
    var porK = {}; cfg.questoes.forEach(function (q) { porK[q.k] = q; });

    // treino aberto do histórico: reconstrói a partir do que ficou salvo
    if (ver && DB.sims[ver]) { var sv = DB.sims[ver]; if (sv.questoes) { cfg.questoes = sv.questoes; porK = {}; cfg.questoes.forEach(function (q) { porK[q.k] = q; }); } return mostraResultado(ver); }
    if (ver && !DB.sims[ver]) { location.replace(location.pathname); return; }

    var id = null;
    Object.keys(DB.sims).forEach(function (k) { var s = DB.sims[k]; if (s.status === "andamento" && s.tipo === cfg.tipo && (cfg.tipo === "prova" ? s.prova === cfg.prova : false)) id = k; });
    if (!id && cfg.tipo === "prova" && !url.searchParams.has("novo")) {
      var fe = sims("prova").filter(function (s) { return s.prova === cfg.prova; }).pop();
      if (fe) return mostraResultado(fe.id);
    }
    if (!id) {
      id = cfg.tipo === "prova" ? "p" + cfg.prova + "-" + Date.now() : cfg.id;
      DB.sims[id] = { tipo: cfg.tipo, prova: cfg.prova || "", titulo: cfg.titulo, filtro: cfg.filtro || null, itens: cfg.questoes.map(function (q) { return q.k; }), respostas: {}, revisar: {}, status: "andamento", inicio: Date.now(), segundos: 0, atual: 0 };
      if (cfg.tipo === "treino") DB.sims[id].questoes = cfg.questoes;
      save();
    }
    var sim = DB.sims[id], qs = cfg.tipo === "treino" && sim.questoes ? sim.questoes : cfg.questoes;
    $("#sim-titulo").textContent = sim.titulo;
    var base = sim.segundos || 0, t0 = Date.now();
    function gasto() { return base + (Date.now() - t0) / 1000; }
    function guarda() { sim.segundos = gasto(); base = sim.segundos; t0 = Date.now(); save(); }
    var timer = $("#timer");
    function tick() { var r = cfg.limite - gasto(); timer.textContent = (r < 0 ? "+" : "") + fmtTempo(Math.abs(r)); timer.className = "timer" + (r < 0 ? " over" : r < 1800 ? " warn" : ""); timer.title = r < 0 ? "Tempo excedido" : "Tempo restante"; }
    tick(); var iv = setInterval(tick, 1000), ivs = setInterval(guarda, 15000);
    document.addEventListener("visibilitychange", function () { if (document.hidden) guarda(); });
    window.addEventListener("pagehide", guarda);

    function grid() {
      $("#qgrid").innerHTML = qs.map(function (q, j) { var k = q.k; return '<button type="button" data-j="' + j + '" class="' + (sim.respostas[k] ? "has " : "") + (sim.revisar[k] ? "flag " : "") + (j === sim.atual ? "cur" : "") + '">' + (cfg.tipo === "prova" ? q.n : j + 1) + "</button>"; }).join("");
      $$("#qgrid button").forEach(function (b) { b.onclick = function () { mostra(+b.dataset.j); }; });
      $("#sim-cont").textContent = Object.keys(sim.respostas).length + "/" + qs.length + " respondidas";
    }
    function imgs(q) {
      var h = ""; (q.c || []).forEach(function (cx) { h += '<p class="ctxlabel">Texto de apoio</p><div class="sheet" data-zoom>' + cx.map(function (s) { return '<img src="' + s + '" alt="Texto de apoio">'; }).join("") + "</div>"; });
      return h + '<div class="sheet" data-zoom>' + q.i.map(function (s) { return '<img src="' + s + '" alt="Questão ' + q.n + '">'; }).join("") + "</div>";
    }
    function mostra(j) {
      j = Math.max(0, Math.min(qs.length - 1, j)); sim.atual = j; guarda();
      var q = qs[j];
      $("#q-tit").textContent = cfg.tipo === "prova" ? "Questão " + q.n : "Questão " + (j + 1) + " de " + qs.length;
      $("#q-orig").textContent = cfg.tipo === "treino" ? "Fatec " + nomeProva(q.p) + " · Q" + q.n : "";
      $("#q-disc").textContent = q.d || "";
      $("#q-img").innerHTML = imgs(q);
      $$(".ans").forEach(function (b) { b.setAttribute("aria-pressed", String(sim.respostas[q.k] === b.dataset.l)); });
      $("#b-prev").disabled = j === 0; $("#b-next").disabled = j === qs.length - 1;
      $("#b-rev").checked = !!sim.revisar[q.k];
      grid(); window.scrollTo(0, 0);
      var prox = qs[j + 1]; if (prox) prox.i.forEach(function (s) { var im = new Image(); im.src = s; });
    }
    function marca(L) {
      var q = qs[sim.atual]; if (sim.respostas[q.k] === L) delete sim.respostas[q.k]; else sim.respostas[q.k] = L;
      guarda(); $$(".ans").forEach(function (b) { b.setAttribute("aria-pressed", String(sim.respostas[q.k] === b.dataset.l)); }); grid();
    }
    $$(".ans").forEach(function (b) { b.onclick = function () { marca(b.dataset.l); }; });
    $("#b-prev").onclick = function () { mostra(sim.atual - 1); };
    $("#b-next").onclick = function () { mostra(sim.atual + 1); };
    $("#b-rev").onchange = function (e) { var k = qs[sim.atual].k; if (e.target.checked) sim.revisar[k] = true; else delete sim.revisar[k]; save(); grid(); };
    var bg = $("#b-grid"), gr = $("#grid");
    bg.onclick = function () { gr.hidden = !gr.hidden; bg.setAttribute("aria-expanded", String(!gr.hidden)); };
    document.addEventListener("keydown", function (e) {
      if ($(".modal,.zoom") || $("#sim-q").hidden || e.ctrlKey || e.metaKey || e.altKey || /INPUT|TEXTAREA|SELECT/.test(document.activeElement.tagName)) return;
      if (/^[a-eA-E]$/.test(e.key)) marca(e.key.toUpperCase());
      else if (e.key === "ArrowRight") mostra(sim.atual + 1);
      else if (e.key === "ArrowLeft") mostra(sim.atual - 1);
    });
    $("#b-fim").onclick = function () {
      var falta = qs.length - Object.keys(sim.respostas).length;
      modal("Finalizar e corrigir?", falta ? "Ainda há " + falta + " questões sem resposta. Em branco conta como erro, como na prova." : "Todas as questões estão respondidas.", "Finalizar", function () {
        guarda(); clearInterval(iv); clearInterval(ivs);
        fetch("/api/corrigir", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ itens: qs.map(function (q) { return q.p + "|" + q.n; }), respostas: chavesServidor() }) })
          .then(function (r) { if (!r.ok) throw new Error(); return r.json(); })
          .then(function (res) { sim.status = "finalizado"; sim.fim = Date.now(); sim.resultado = res; save(); mostraResultado(id); })
          .catch(function () { toast("Não consegui corrigir agora. O portal está rodando?"); iv = setInterval(tick, 1000); });
      });
    };
    function chavesServidor() { var o = {}; qs.forEach(function (q) { if (sim.respostas[q.k]) o[q.p + "|" + q.n] = sim.respostas[q.k]; }); return o; }
    mostra(sim.atual || 0);

    function mostraResultado(sid) {
      var s = DB.sims[sid], r = s.resultado, lista = s.questoes || cfg.questoes;
      clearInterval(iv); clearInterval(ivs);
      $(".simbar").hidden = true; $("#sim-q").hidden = true; $("#sim-res").hidden = false;
      $("#res-eyebrow").textContent = (s.tipo === "prova" ? "Resultado do simulado" : "Resultado do treino") + " · " + fmtData(s.fim);
      $("#res-tit").textContent = s.titulo;
      var red = s.tipo === "prova" && DB.reds[s.prova] && DB.reds[s.prova].nota != null ? DB.reds[s.prova].nota : null;
      $("#res-stats").innerHTML = '<div class="stat"><b>' + r.acertos + "/" + r.total + '</b><span>acertos</span></div><div class="stat"><b>' + r.nota + '</b><span>nota objetiva (0–100)</span></div><div class="stat"><b>' + fmtTempo(s.segundos || 0) + "</b><span>tempo usado</span></div>" +
        (s.tipo === "prova" ? '<div class="stat"><b>' + (red == null ? "–" : Math.round(r.nota * 0.8 + red * 0.2)) + "</b><span>" + (red == null ? "nota final (falta a nota da redação)" : "nota final estimada (80/20)") + "</span></div>" : "");
      $("#res-disc").innerHTML = barras(r.porDisc, DISC);
      var ruins = Object.keys(r.porTema).filter(function (t) { return r.porTema[t][0] < r.porTema[t][1]; }).sort(function (a, b) { return (r.porTema[b][1] - r.porTema[b][0]) - (r.porTema[a][1] - r.porTema[a][0]); }).slice(0, 12);
      $("#res-temas-wrap").hidden = !ruins.length; $("#res-temas").innerHTML = barras(r.porTema, ruins);
      var porChave = {}; (r.itens || []).forEach(function (it) { porChave[it.k] = it; });
      $("#res-grid").innerHTML = lista.map(function (q, j) { var it = porChave[q.p + "|" + q.n] || {}; return '<button type="button" data-j="' + j + '" class="' + (it.ok ? "right" : "wrong") + '">' + (s.tipo === "prova" ? q.n : j + 1) + "</button>"; }).join("");
      $$("#res-grid button").forEach(function (b) {
        b.onclick = function () {
          var q = lista[+b.dataset.j], it = porChave[q.p + "|" + q.n] || {}, dado = s.respostas[q.k] || "em branco", anul = /ANUL/i.test(it.gab || "");
          $("#res-q").innerHTML = "<h3>Questão " + q.n + (s.tipo === "treino" ? " · Fatec " + nomeProva(q.p) : "") + '</h3><p class="disp small">Você: <b>' + esc(dado) + "</b> · Gabarito: <b>" + esc(it.gab) + "</b>" + (anul ? " (anulada, conta para todos)" : "") + " · " + esc(it.tema || "") + (it.sub ? " — " + esc(it.sub) : "") + "</p>" + imgs(q);
          $("#res-q").scrollIntoView({ behavior: "smooth", block: "start" });
        };
      });
      $("#prompt-rev").value = cfg.prompt;
      var form = $("#rev-form");
      form.onsubmit = function () { $("#rev-dados").value = JSON.stringify({ titulo: "Revisão — " + s.titulo, itens: lista.map(function (q) { return q.p + "|" + q.n; }), respostas: (function () { var o = {}; lista.forEach(function (q) { if (s.respostas[q.k]) o[q.p + "|" + q.n] = s.respostas[q.k]; }); return o; })() }); };
      if (s.tipo === "prova") { var a = $("#res-red"); a.hidden = false; a.href = "/redacao/" + s.prova; }
      $("#res-del").onclick = function () { modal("Apagar este resultado?", "O simulado some do histórico e do caderno de erros. Não dá para desfazer.", "Apagar", function () { delete DB.sims[sid]; save(); location.href = "/simulados"; }); };
      if (s.tipo === "prova") {
        var refaz = document.createElement("a"); refaz.className = "btn ghost"; refaz.href = "/simulado/" + s.prova + "?novo=1"; refaz.textContent = "Refazer esta prova";
        $("#res-del").parentNode.insertBefore(refaz, $("#res-del").previousElementSibling);
      }
      window.scrollTo(0, 0);
    }
  }

  // ---------- CADERNO DE ERROS ----------
  var erros = $("#erros");
  if (erros) {
    var mapa = {}; // chave -> {tema, quando, dado}
    sims().forEach(function (s) {
      var resp = {}; (s.questoes || []).forEach(function (q) { if (s.respostas[q.k]) resp[q.p + "|" + q.n] = s.respostas[q.k]; });
      if (s.tipo === "prova") Object.keys(s.respostas).forEach(function (k) { resp[s.prova + "|" + k] = s.respostas[k]; });
      (s.resultado.itens || []).forEach(function (it) { if (!it.ok) mapa[it.k] = { tema: it.tema || "Outro", dado: resp[it.k] || "em branco", gab: it.gab, sub: it.sub }; });
    });
    var ks = Object.keys(mapa);
    if (ks.length) {
      fetch("/api/questoes?k=" + encodeURIComponent(ks.join(","))).then(function (r) { return r.json(); }).then(function (qs) {
        var porTema = {}; (qs || []).forEach(function (q) { var m = mapa[q.k]; (porTema[m.tema] = porTema[m.tema] || []).push({ q: q, m: m }); });
        var temas = Object.keys(porTema).sort(function (a, b) { return porTema[b].length - porTema[a].length; });
        $("#e-lista").innerHTML = temas.map(function (t) {
          var l = porTema[t];
          return '<h2>' + esc(t) + ' <span class="pill bad">' + l.length + '</span></h2><a class="btn small" href="/treino?tema=' + encodeURIComponent(t) + '">Treinar este tema</a>' +
            l.slice(0, 10).map(function (x) {
              var q = x.q, h = (q.c || []).map(function (cx) { return '<p class="ctxlabel">Texto de apoio</p><div class="sheet" data-zoom>' + cx.map(function (s) { return '<img src="' + s + '" alt="Texto de apoio" loading="lazy">'; }).join("") + "</div>"; }).join("");
              return "<h3>Fatec " + nomeProva(q.p) + " · questão " + q.n + ' <span class="small muted disp">você: ' + esc(x.m.dado) + " · gabarito: " + esc(q.gab) + "</span></h3>" + (q.sub ? '<p class="small muted">' + esc(q.sub) + "</p>" : "") + h + '<div class="sheet" data-zoom>' + q.i.map(function (s) { return '<img src="' + s + '" alt="Questão ' + q.n + '" loading="lazy">'; }).join("") + "</div>";
            }).join("") + (l.length > 10 ? '<p class="small muted">+ ' + (l.length - 10) + " questões deste tema.</p>" : "");
        }).join("");
      });
    }
  }

  // ---------- DESEMPENHO ----------
  function grafico(pts, rotulo) {
    var W = 640, H = 220, L = 34, R = 14, T = 16, B = 34, n = pts.length;
    var X = function (i) { return n < 2 ? L + (W - L - R) / 2 : L + i * (W - L - R) / (n - 1); }, Y = function (v) { return T + (100 - v) * (H - T - B) / 100; };
    var g = [0, 25, 50, 75, 100].map(function (v) { return '<line class="grid" x1="' + L + '" x2="' + (W - R) + '" y1="' + Y(v) + '" y2="' + Y(v) + '"/><text x="' + (L - 6) + '" y="' + (Y(v) + 4) + '" text-anchor="end">' + v + "</text>"; }).join("");
    var d = pts.map(function (p, i) { return (i ? "L" : "M") + X(i) + " " + Y(p.y); }).join(" ");
    var area = n > 1 ? '<path class="area" d="' + d + " L" + X(n - 1) + " " + Y(0) + " L" + X(0) + " " + Y(0) + ' Z"/>' : "";
    var lab = pts.map(function (p, i) { return '<text x="' + X(i) + '" y="' + (H - 12) + '" text-anchor="middle">' + esc(p.x) + "</text>"; }).join("");
    var dots = pts.map(function (p, i) { return '<circle class="dot" cx="' + X(i) + '" cy="' + Y(p.y) + '" r="' + (i === n - 1 ? 5 : 3.5) + '"/>' + (i === n - 1 ? '<text class="lbl" x="' + X(i) + '" y="' + (Y(p.y) - 10) + '" text-anchor="middle">' + p.y + "</text>" : ""); }).join("");
    return '<svg class="chart" viewBox="0 0 ' + W + " " + H + '" role="img" aria-label="' + esc(rotulo) + '">' + g + area + (n > 1 ? '<path class="line" d="' + d + '"/>' : "") + dots + lab + "</svg>";
  }
  var desemp = $("#desempenho");
  if (desemp) {
    var todos = sims(), provas = sims("prova");
    if (todos.length) {
      $("#d-vazio").hidden = true; $("#d-conteudo").hidden = false;
      if (provas.length) {
        $("#d-chart").innerHTML = grafico(provas.map(function (s) { return { x: nomeProva(s.prova), y: s.resultado.nota }; }), "Evolução da nota objetiva");
        var u = provas[provas.length - 1], rn = DB.reds[u.prova] && DB.reds[u.prova].nota;
        $("#d-stats").innerHTML = '<div class="stat"><b>' + u.resultado.nota + '</b><span>último simulado</span></div><div class="stat"><b>' + Math.round(provas.reduce(function (a, s) { return a + s.resultado.nota; }, 0) / provas.length) + '</b><span>média dos simulados</span></div><div class="stat"><b>' + (rn == null ? "–" : Math.round(u.resultado.nota * 0.8 + rn * 0.2)) + "</b><span>nota final estimada (80/20)</span></div>";
      } else $("#d-sim").hidden = true;
      var ag = {}, tm = {};
      todos.forEach(function (s) {
        Object.keys(s.resultado.porDisc).forEach(function (d) { ag[d] = ag[d] || [0, 0]; ag[d][0] += s.resultado.porDisc[d][0]; ag[d][1] += s.resultado.porDisc[d][1]; });
        Object.keys(s.resultado.porTema).forEach(function (t) { tm[t] = tm[t] || [0, 0]; tm[t][0] += s.resultado.porTema[t][0]; tm[t][1] += s.resultado.porTema[t][1]; });
      });
      $("#d-disc").innerHTML = barras(ag, DISC);
      var fr = Object.keys(tm).filter(function (t) { return tm[t][1] >= 2; }).sort(function (a, b) { return tm[a][0] / tm[a][1] - tm[b][0] / tm[b][1] || tm[b][1] - tm[a][1]; }).slice(0, 10);
      $("#d-fracos-w").hidden = !fr.length; $("#d-fracos").innerHTML = barras(tm, fr);
    }
    var rs = Object.keys(DB.reds).filter(function (k) { return DB.reds[k].nota != null; }).map(function (k) { return { x: nomeProva(k), y: DB.reds[k].nota, t: DB.reds[k].notaEm }; }).sort(function (a, b) { return a.t - b.t; });
    if (rs.length) { $("#d-vazio").hidden = true; $("#d-conteudo").hidden = false; $("#d-red").innerHTML = grafico(rs, "Notas de redação"); } else $("#d-red-w").hidden = true;
  }

  // ---------- REDAÇÃO ----------
  $$("[data-red]").forEach(function (li) { var r = DB.reds[li.dataset.red]; if (r && r.pdfEm) $("[data-transcrita]", li).hidden = false; });
  var red = $("#redacao");
  if (red) {
    var pv = red.dataset.prova, R = DB.reds[pv] = DB.reds[pv] || { titulo: "", texto: "" };
    var rt = $("#rt"), rx = $("#rx"), rc = $("#rc"), deb;
    rt.value = R.titulo || ""; rx.value = R.texto || "";
    if (R.nota != null) $("#red-nota").value = R.nota;
    var conta2 = function () {
      var t = rx.value, pal = (t.trim().match(/\S+/g) || []).length;
      var lin = t.trim() ? t.split(/\n/).filter(function (l) { return l.trim(); }).reduce(function (a, l) { return a + Math.max(1, Math.ceil(l.length / 75)); }, 0) : 0;
      rc.textContent = pal + " palavras · ≈ " + lin + " linhas manuscritas (estimativa: 75 caracteres por linha)";
      return lin;
    };
    var agenda = function () { conta2(); clearTimeout(deb); deb = setTimeout(function () { R.titulo = rt.value; R.texto = rx.value; R.atualizado = Date.now(); save(); }, 600); };
    rt.oninput = agenda; rx.oninput = agenda; conta2();
    $("#red-form").addEventListener("submit", function (e) {
      R.titulo = rt.value; R.texto = rx.value; save();
      if (!rt.value.trim()) { e.preventDefault(); toast("Falta o título: sem título a Fatec desconta."); rt.focus(); return; }
      if (conta2() <= 5) { e.preventDefault(); toast("Texto muito curto: 5 linhas ou menos zera a redação."); rx.focus(); return; }
      R.pdfEm = Date.now(); save(); toast("PDF gerado. Agora copie o prompt e mande os dois para a IA.");
    });
    $("#red-salvar-nota").onclick = function () {
      var v = parseInt($("#red-nota").value, 10);
      if (isNaN(v) || v < 0 || v > 100) { $("#red-nota-msg").textContent = "Use um número de 0 a 100."; return; }
      R.nota = v; R.notaEm = Date.now(); save(); $("#red-nota-msg").textContent = "Nota salva.";
    };
  }

  // ---------- AUTORES ----------
  var fa = $("#filtro-autor");
  if (fa) fa.oninput = function () { var q = fa.value.toLowerCase(); $$("#autores > li").forEach(function (li) { li.hidden = q && li.dataset.busca.toLowerCase().indexOf(q) < 0; }); };

  // ---------- DADOS ----------
  var ex1 = $("#exportar");
  if (ex1) {
    ex1.onclick = function () {
      var b = new Blob([JSON.stringify(DB, null, 1)], { type: "application/json" }), a = document.createElement("a");
      a.href = URL.createObjectURL(b); a.download = "fatec-turbo-progresso-" + new Date().toISOString().slice(0, 10) + ".json"; document.body.appendChild(a); a.click(); a.remove();
      $("#dados-msg").textContent = "Arquivo exportado.";
    };
    $("#importar").onchange = function (e) {
      var f = e.target.files[0]; if (!f) return; var rd = new FileReader();
      rd.onload = function () {
        try { var d = JSON.parse(rd.result); if (!d || d.v !== 1) throw 0;
          modal("Importar este arquivo?", "O progresso deste navegador será substituído pelo do arquivo.", "Importar", function () { DB = d; save(); $("#dados-msg").textContent = "Progresso importado."; });
        } catch (x) { $("#dados-msg").textContent = "Esse arquivo não é um backup do Fatec Turbo."; }
      };
      rd.readAsText(f);
    };
    $("#apagar").onclick = function () { modal("Apagar todo o progresso?", "Simulados, lições, rotina e redações deste navegador serão removidos. Não dá para desfazer.", "Apagar tudo", function () { DB = vazio(); save(); $("#dados-msg").textContent = "Progresso apagado."; }); };
  }
})();
