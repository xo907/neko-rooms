// Room extras: reactions, raised hands, the up next queue and what is
// playing. Injected into neko room pages, talks to the live channel of the
// room with the ticket that came with the join link.
(function () {
  var C = __NR_CONFIG__;
  var KEY = 'nr-ticket:' + C.room;
  var params = new URLSearchParams(location.search);
  var ticket = params.get('nrt');
  try {
    if (ticket) sessionStorage.setItem(KEY, ticket);
    else ticket = sessionStorage.getItem(KEY);
  } catch (e) { /* storage blocked */ }
  if (!ticket || !window.EventSource) return;

  var base = C.api + 'public/live/' + encodeURIComponent(C.room) + '/';
  var me = null, queue = [], hands = [], playing = '';
  var ui = {};

  //
  // neko client
  //

  function vm() { var el = document.getElementById('neko'); return el && el.__vue__; }
  function acc() { var v = vm(); return v && v.$accessor; }
  function myName() {
    try { var n = acc().user.member.displayname; if (n) return n; } catch (e) { /* not connected */ }
    return params.get('usr') || '';
  }
  function hosting() { try { return !!acc().remote.hosting; } catch (e) { return false; } }
  function hostName() {
    try { var a = acc(), m = a.user.members[a.remote.id]; return m ? m.displayname : ''; } catch (e) { return ''; }
  }
  function memberByName(name) {
    try {
      var ms = acc().user.members;
      for (var id in ms) if (ms[id] && ms[id].displayname === name) return ms[id];
    } catch (e) { /* not connected */ }
    return null;
  }

  //
  // server
  //

  function post(path, body) {
    body = body || {};
    body.t = ticket;
    body.name = myName();
    return fetch(base + path, {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json', 'X-Requested-With': 'XMLHttpRequest' },
      body: JSON.stringify(body),
    }).then(function (r) {
      if (!r.ok) return r.text().then(function (t) { throw new Error((t || 'Error ' + r.status).trim()); });
    });
  }

  function connect() {
    var es = new EventSource(base + 'events?t=' + encodeURIComponent(ticket) + '&name=' + encodeURIComponent(myName()));
    es.onmessage = function (e) {
      var ev;
      try { ev = JSON.parse(e.data); } catch (err) { return; }
      handle(ev);
    };
  }

  function handle(ev) {
    switch (ev.type) {
      case 'init':
        me = ev.you;
        queue = ev.queue || [];
        hands = ev.hands || [];
        playing = ev.playing || '';
        render();
        break;
      case 'queue': queue = ev.queue || []; renderQueue(); break;
      case 'hands': hands = ev.hands || []; renderHands(); break;
      case 'playing': playing = ev.title || ''; renderPlaying(); break;
      case 'reaction': fly(ev.emoji, ev.name); break;
      case 'played': toast((ev.name || 'Someone') + ' played ' + (ev.item.title || host(ev.item.url))); break;
    }
  }

  //
  // view
  //

  function el(tag, cls, text) {
    var e = document.createElement(tag);
    if (cls) e.className = cls;
    if (text != null) e.textContent = text;
    return e;
  }
  function host(u) { try { return new URL(u).hostname.replace(/^www\./, ''); } catch (e) { return u; } }

  var css = '' +
    '.nrx{--a:var(--nr-primary,#2dd4bf);--s:rgba(14,14,17,.78);--l:rgba(255,255,255,.1);font:500 13px/1.4 var(--nr-font,system-ui,sans-serif);color:#fafafa}' +
    '.nrx-dock{position:absolute;left:14px;bottom:14px;z-index:30;display:flex;gap:6px;align-items:center}' +
    '.nrx-btn{position:relative;display:inline-flex;align-items:center;justify-content:center;gap:6px;height:36px;min-width:36px;padding:0 12px;border:0;border-radius:99px;background:var(--s);color:inherit;font:inherit;cursor:pointer;backdrop-filter:blur(12px);box-shadow:inset 0 0 0 1px var(--l);transition:background .18s,transform .18s}' +
    '.nrx-btn:hover{background:rgba(32,32,38,.9)}.nrx-btn:active{transform:scale(.96)}' +
    '.nrx-btn.on{background:var(--a);color:#06110f}' +
    '.nrx-badge{position:absolute;top:-4px;right:-4px;min-width:18px;height:18px;padding:0 5px;border-radius:9px;background:var(--a);color:#06110f;font-size:11px;line-height:18px;text-align:center}' +
    '.nrx-emojis{display:none;gap:2px;padding:4px;border-radius:99px;background:var(--s);backdrop-filter:blur(12px);box-shadow:inset 0 0 0 1px var(--l)}' +
    '.nrx-emojis.open{display:flex}' +
    '.nrx-emojis button{width:34px;height:34px;border:0;border-radius:50%;background:transparent;font-size:19px;cursor:pointer;transition:transform .15s,background .15s}' +
    '.nrx-emojis button:hover{background:rgba(255,255,255,.1);transform:scale(1.18)}' +
    '.nrx-fly{position:absolute;bottom:60px;z-index:29;pointer-events:none;display:flex;flex-direction:column;align-items:center;animation:nrx-up 3.2s cubic-bezier(.2,.6,.3,1) forwards}' +
    '.nrx-fly b{font-size:34px;line-height:1;filter:drop-shadow(0 4px 10px rgba(0,0,0,.5))}' +
    '.nrx-fly span{margin-top:2px;padding:1px 7px;border-radius:99px;background:rgba(0,0,0,.55);font-size:11px;white-space:nowrap}' +
    '@keyframes nrx-up{0%{transform:translate(0,0) scale(.6);opacity:0}10%{opacity:1;transform:translate(0,-20px) scale(1.1)}100%{transform:translate(var(--dx),-340px) scale(1);opacity:0}}' +
    '.nrx-top{position:absolute;left:14px;top:14px;z-index:30;display:flex;flex-direction:column;gap:6px;align-items:flex-start;max-width:min(420px,70%)}' +
    '.nrx-chip{display:flex;align-items:center;gap:8px;padding:6px 6px 6px 12px;border-radius:99px;background:var(--s);backdrop-filter:blur(12px);box-shadow:inset 0 0 0 1px var(--l);max-width:100%}' +
    '.nrx-chip.solo{padding-right:12px}' +
    '.nrx-chip em{font-style:normal;color:rgba(255,255,255,.6)}' +
    '.nrx-chip .t{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}' +
    '.nrx-chip button{height:26px;padding:0 10px;border:0;border-radius:99px;background:var(--a);color:#06110f;font:600 12px/1 inherit;cursor:pointer;white-space:nowrap}' +
    '.nrx-chip button.ghost{background:rgba(255,255,255,.1);color:#fafafa}' +
    '.nrx-panel{position:absolute;left:14px;bottom:60px;z-index:31;width:min(360px,calc(100% - 28px));max-height:min(60%,460px);display:none;flex-direction:column;border-radius:16px;background:rgba(14,14,17,.92);backdrop-filter:blur(16px);box-shadow:inset 0 0 0 1px var(--l),0 30px 70px -20px rgba(0,0,0,.8)}' +
    '.nrx-panel.open{display:flex}' +
    '.nrx-panel h4{margin:0;padding:14px 16px 10px;font-size:14px;font-weight:600;display:flex;justify-content:space-between;align-items:center}' +
    '.nrx-panel h4 em{font-style:normal;font-weight:500;color:rgba(255,255,255,.5);font-size:12px}' +
    '.nrx-add{display:flex;gap:6px;padding:0 12px 10px}' +
    '.nrx-add input{flex:1;min-width:0;height:34px;padding:0 12px;border:1px solid var(--l);border-radius:10px;background:rgba(255,255,255,.05);color:#fafafa;font:inherit;outline:none}' +
    '.nrx-add input:focus{border-color:var(--a)}' +
    '.nrx-add button{height:34px;padding:0 14px;border:0;border-radius:10px;background:var(--a);color:#06110f;font:600 13px/1 inherit;cursor:pointer}' +
    '.nrx-list{list-style:none;margin:0;padding:0 8px 8px;overflow:auto}' +
    '.nrx-list li{display:flex;align-items:center;gap:8px;padding:8px;border-radius:10px}' +
    '.nrx-list li:hover{background:rgba(255,255,255,.05)}' +
    '.nrx-vote{display:flex;flex-direction:column;align-items:center;min-width:34px;height:40px;justify-content:center;border:0;border-radius:10px;background:rgba(255,255,255,.06);color:#fafafa;font:600 12px/1.1 inherit;cursor:pointer}' +
    '.nrx-vote.on{background:color-mix(in srgb,var(--a) 22%,transparent);color:var(--a)}' +
    '.nrx-item{flex:1;min-width:0}' +
    '.nrx-item a{display:block;color:#fafafa;text-decoration:none;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}' +
    '.nrx-item small{display:block;color:rgba(255,255,255,.5);font-size:11.5px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}' +
    '.nrx-icon{width:30px;height:30px;border:0;border-radius:8px;background:transparent;color:rgba(255,255,255,.7);font-size:14px;cursor:pointer}' +
    '.nrx-icon:hover{background:rgba(255,255,255,.1);color:#fff}' +
    '.nrx-icon.play{color:var(--a)}' +
    '.nrx-empty{padding:4px 16px 16px;color:rgba(255,255,255,.5)}' +
    '.nrx-toast{position:absolute;left:50%;bottom:64px;z-index:32;transform:translateX(-50%);padding:8px 14px;border-radius:99px;background:rgba(14,14,17,.92);box-shadow:inset 0 0 0 1px var(--l);white-space:nowrap;max-width:80%;overflow:hidden;text-overflow:ellipsis;animation:nrx-fade 3.5s forwards}' +
    '@keyframes nrx-fade{0%,85%{opacity:1}100%{opacity:0}}' +
    '@media (max-width:600px){.nrx-btn span.l{display:none}.nrx-dock{left:8px;bottom:8px}.nrx-top{left:8px;top:8px}}';

  function mount() {
    var host = document.querySelector('#neko .video-container') || document.querySelector('#neko .video');
    if (!host) return false;
    if (getComputedStyle(host).position === 'static') host.style.position = 'relative';

    var style = el('style');
    style.textContent = css;
    document.head.appendChild(style);

    var root = el('div', 'nrx');
    ui.root = root;

    ui.top = el('div', 'nrx-top');
    ui.playing = el('div', 'nrx-chip solo');
    ui.hands = el('div');
    ui.hands.style.cssText = 'display:flex;flex-direction:column;gap:6px';
    ui.top.appendChild(ui.playing);
    ui.top.appendChild(ui.hands);

    var dock = el('div', 'nrx-dock');
    ui.react = el('button', 'nrx-btn');
    ui.react.title = 'React';
    ui.react.innerHTML = '<span>😀</span>';
    ui.emojis = el('div', 'nrx-emojis');
    C.reactions.forEach(function (e) {
      var b = el('button', '', e);
      b.title = e;
      b.onclick = function () { post('react', { emoji: e }).catch(function () {}); };
      ui.emojis.appendChild(b);
    });
    ui.react.onclick = function () { ui.emojis.classList.toggle('open'); };

    ui.hand = el('button', 'nrx-btn');
    ui.hand.title = 'Raise your hand to get control';
    ui.hand.onclick = function () {
      var up = !mine();
      post('hand', { up: up }).catch(function (e) { toast(e.message); });
    };

    ui.next = el('button', 'nrx-btn');
    ui.next.title = 'Up next';
    ui.next.onclick = function () {
      ui.panel.classList.toggle('open');
      if (ui.panel.classList.contains('open')) ui.input.focus();
    };

    dock.appendChild(ui.react);
    dock.appendChild(ui.emojis);
    dock.appendChild(ui.hand);
    dock.appendChild(ui.next);

    ui.panel = el('div', 'nrx-panel');
    var h = el('h4');
    h.appendChild(el('span', '', 'Up next'));
    ui.count = el('em');
    h.appendChild(ui.count);
    ui.panel.appendChild(h);
    var form = el('form', 'nrx-add');
    ui.input = el('input');
    ui.input.placeholder = 'Paste a link to add it';
    ui.input.setAttribute('autocomplete', 'off');
    var add = el('button', '', 'Add');
    add.type = 'submit';
    form.appendChild(ui.input);
    form.appendChild(add);
    form.onsubmit = function (e) {
      e.preventDefault();
      var url = ui.input.value.trim();
      if (!url) return;
      post('queue', { url: url }).then(function () { ui.input.value = ''; }).catch(function (err) { toast(err.message); });
    };
    // keys typed here must not reach the remote browser
    ['keydown', 'keyup', 'keypress'].forEach(function (t) {
      ui.panel.addEventListener(t, function (e) { e.stopPropagation(); });
    });
    ui.panel.appendChild(form);
    ui.list = el('ul', 'nrx-list');
    ui.panel.appendChild(ui.list);

    root.appendChild(ui.top);
    root.appendChild(dock);
    root.appendChild(ui.panel);
    host.appendChild(root);
    ui.host = host;
    return true;
  }

  function mine() {
    if (!me) return false;
    for (var i = 0; i < hands.length; i++) if (hands[i].id === me.id) return true;
    return false;
  }

  function render() {
    renderPlaying();
    renderHands();
    renderQueue();
  }

  function renderPlaying() {
    if (!ui.playing) return;
    ui.playing.innerHTML = '';
    if (!playing) { ui.playing.style.display = 'none'; return; }
    ui.playing.style.display = '';
    ui.playing.appendChild(el('em', '', 'Now playing'));
    ui.playing.appendChild(el('span', 't', playing));
  }

  function renderHands() {
    if (!ui.hand) return;
    var up = mine();
    ui.hand.className = 'nrx-btn' + (up ? ' on' : '');
    ui.hand.innerHTML = '<span>✋</span><span class="l">' + (up ? 'Lower hand' : 'Raise hand') + '</span>';
    if (hands.length) {
      var b = el('span', 'nrx-badge', String(hands.length));
      ui.hand.appendChild(b);
    }

    ui.hands.innerHTML = '';
    var giving = hosting();
    hands.forEach(function (h, i) {
      var chip = el('div', 'nrx-chip' + (giving ? '' : ' solo'));
      chip.appendChild(el('span', '', '✋'));
      chip.appendChild(el('span', 't', h.name + (me && h.id === me.id ? ' (you)' : '')));
      if (i === 0 && !giving) chip.appendChild(el('em', '', 'is next'));
      if (giving && (!me || h.id !== me.id)) {
        var give = el('button', '', 'Give control');
        give.onclick = function () {
          var m = memberByName(h.name);
          try {
            if (m) acc().remote.give(m);
            else acc().remote.release();
          } catch (e) { /* not connected */ }
          post('hand', { up: false, target: h.id }).catch(function () {});
        };
        chip.appendChild(give);
      }
      ui.hands.appendChild(chip);
    });
  }

  function renderQueue() {
    if (!ui.next) return;
    ui.next.innerHTML = '<span>▶</span><span class="l">Up next</span>';
    if (queue.length) ui.next.appendChild(el('span', 'nrx-badge', String(queue.length)));
    ui.count.textContent = queue.length ? queue.length + (queue.length === 1 ? ' link' : ' links') : '';

    ui.list.innerHTML = '';
    if (!queue.length) {
      var empty = el('li', 'nrx-empty', 'Nothing queued. Add a video or page everyone can vote on.');
      ui.list.appendChild(empty);
      return;
    }
    var control = hostName();
    queue.forEach(function (it) {
      var li = el('li');
      var voted = me && it.voters.indexOf(me.id) >= 0;
      var vote = el('button', 'nrx-vote' + (voted ? ' on' : ''));
      vote.title = voted ? 'Take back your vote' : 'Vote for it';
      vote.appendChild(el('span', '', '▲'));
      vote.appendChild(el('span', '', String(it.voters.length)));
      vote.onclick = function () { post('queue/' + it.id + '/vote').catch(function (e) { toast(e.message); }); };
      li.appendChild(vote);

      var info = el('div', 'nrx-item');
      var a = el('a', '', it.title || host(it.url));
      a.href = it.url;
      a.target = '_blank';
      a.rel = 'noopener noreferrer';
      info.appendChild(a);
      info.appendChild(el('small', '', host(it.url) + ' · added by ' + it.by));
      li.appendChild(info);

      if (!control || control === myName() || (me && me.manager)) {
        var play = el('button', 'nrx-icon play', '▶');
        play.title = 'Play it now';
        play.onclick = function () { post('queue/' + it.id + '/play').catch(function (e) { toast(e.message); }); };
        li.appendChild(play);
      }
      if (me && (it.by_id === me.id || me.manager || control === myName())) {
        var rm = el('button', 'nrx-icon', '✕');
        rm.title = 'Remove';
        rm.onclick = function () { post('queue/' + it.id + '/remove').catch(function (e) { toast(e.message); }); };
        li.appendChild(rm);
      }
      ui.list.appendChild(li);
    });
  }

  function fly(emoji, name) {
    if (!ui.root) return;
    var f = el('div', 'nrx-fly');
    f.appendChild(el('b', '', emoji));
    if (name) f.appendChild(el('span', '', name));
    var w = ui.host.clientWidth || 600;
    f.style.left = Math.round(40 + Math.random() * Math.min(260, w - 120)) + 'px';
    f.style.setProperty('--dx', Math.round((Math.random() - 0.5) * 120) + 'px');
    ui.root.appendChild(f);
    setTimeout(function () { f.remove(); }, 3300);
  }

  function toast(text) {
    if (!ui.root || !text) return;
    var t = el('div', 'nrx-toast', text);
    ui.root.appendChild(t);
    setTimeout(function () { t.remove(); }, 3600);
  }

  //
  // start once the neko client is connected, so names are known
  //

  var tries = 0;
  var timer = setInterval(function () {
    tries++;
    var a = acc();
    var ready = a && a.user && a.user.member && a.user.member.displayname;
    if (!ready && tries < 30) return;
    if (!ui.root && !mount()) { if (tries < 60) return; }
    clearInterval(timer);
    if (!ui.root) return;
    connect();

    // hands follow control: getting it lowers your hand, the person in
    // control sees the give buttons
    var v = vm();
    if (v && v.$store) {
      v.$store.watch(function (s) { return s.remote && s.remote.id; }, function () {
        if (hosting() && mine()) post('hand', { up: false }).catch(function () {});
        renderHands();
        renderQueue();
      });
    }
  }, 1000);
})();
