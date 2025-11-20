const $=s=>document.querySelector(s)
const $$=s=>document.querySelectorAll(s)
const setTheme=t=>{document.body.setAttribute('data-theme',t)}
const getTheme=()=>localStorage.getItem('mb_theme')||'dark'
const saveTheme=t=>localStorage.setItem('mb_theme',t)
const isAuth=()=>!!localStorage.getItem('mb_auth')
const login=()=>{localStorage.setItem('mb_auth','1')}
const logout=()=>{localStorage.removeItem('mb_auth')}
// removed onboard welcome

function route(){
  const hash=location.hash||'#/home'
  const [_,root,sub]=hash.split('/')
  $$('section.view').forEach(v=>v.classList.add('hidden'))
  $('#'+root)?.classList.remove('hidden')
  if(root==='console'&&!isAuth()){location.hash='#/login';return}
  if(root==='docs') updateDoc(sub||'overview')
  if(root==='console') updateConsole(sub||'dashboard')
  $('#loginLink').classList.toggle('hidden',isAuth())
  $('#logoutBtn').classList.toggle('hidden',!isAuth())
}

function updateDoc(id){
  const titles={overview:'总览',start:'快速开始',architecture:'架构',nrpc:'NRPC',storage:'存储',security:'安全',deploy:'部署',config:'配置',multitenancy:'多租户',rls:'行级安全',console:'控制台使用',api:'API 参考'}
  $('#docTitle').textContent=titles[id]||'文档'
  fetch(`/md/${id}`)
    .then(r=>r.ok?r.text():Promise.resolve('<p>文档尚未准备。</p>'))
    .then(html=>{$('#docBody').innerHTML=html})
}

function updateConsole(id){
  $('#consoleTitle').textContent=$(`[data-console][href="#/console/${id}"]`)?.textContent||'控制台'
  ;['dashboard','requests','analytics','perf','config','tables'].forEach(x=>$('#'+x).classList.toggle('hidden',x!==id))
}

function drawSpark(){
  const c=$('#spark');if(!c)return;const ctx=c.getContext('2d');const w=c.width,h=c.height
  ctx.clearRect(0,0,w,h)
  const points=Array.from({length:60},(_,i)=>({x:i*(w/59),y:h*.6-Math.sin(i/5)*18-(Math.random()*24-12)}))
  ctx.lineWidth=2;ctx.strokeStyle=getComputedStyle(document.body).getPropertyValue('--accent')
  ctx.beginPath();ctx.moveTo(points[0].x,points[0].y);points.forEach(p=>ctx.lineTo(p.x,p.y));ctx.stroke()
}

function genRequests(){
  const body=$('#requestRows');if(!body)return;body.innerHTML=''
  const methods=['GET','POST','PUT','DELETE']
  for(let i=0;i<12;i++){
    const row=document.createElement('div');row.className='table-row'
    const t=new Date(Date.now()-i*60000).toLocaleTimeString()
    const m=methods[Math.floor(Math.random()*methods.length)]
    const p=['/api/users','/api/orders','/rpc/task','/health'][Math.floor(Math.random()*4)]
    const d=(Math.random()*300+20)|0;const s=[200,201,400,404,500][Math.floor(Math.random()*5)]
    row.innerHTML=`<span>${t}</span><span>${m}</span><span>${p}</span><span>${d} ms</span><span>${s}</span>`
    body.appendChild(row)
  }
}

function genTables(){
  const body=$('#tableRows');if(!body)return;body.innerHTML=''
  const data=[['users',1280,'开启'],['orders',542,'开启'],['tasks',87,'关闭'],['logs',15420,'关闭']]
  data.forEach(([name,count,rls])=>{
    const row=document.createElement('div');row.className='table-row'
    row.innerHTML=`<span>${name}</span><span>${count}</span><span>${rls}</span><span><a href="#">打开</a></span>`
    body.appendChild(row)
  })
}

function refreshMetrics(){
  const q=$('#qps'),l=$('#latency'),e=$('#error')
  if(q) q.textContent=(Math.random()*1200+200|0).toString()
  if(l) l.textContent=`${(Math.random()*80+10|0)} ms`
  if(e) e.textContent=`${(Math.random()*3).toFixed(1)}%`
  drawSpark();genRequests();genTables()
}

function bind(){
  window.addEventListener('hashchange',route)
  $('#themeToggle').addEventListener('click',()=>{const next=document.body.getAttribute('data-theme')==='dark'?'light':'dark';setTheme(next);saveTheme(next)})
  $('#loginForm')?.addEventListener('submit',e=>{e.preventDefault();login();location.hash='#/console'})
  $('#logoutBtn').addEventListener('click',()=>{logout();location.hash='#/home'})
  $('#refreshBtn').addEventListener('click',refreshMetrics)
  $$('.main-nav a').forEach(a=>a.addEventListener('click',e=>e.currentTarget.blur()))
}

// welcome removed

function init(){
  setTheme(getTheme())
  bind();route();refreshMetrics()
}

document.addEventListener('DOMContentLoaded',init)