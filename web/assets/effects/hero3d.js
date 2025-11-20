;(function(){
  const fx=new URLSearchParams(location.search).get('fx');if(fx!=='on')return
  const c=document.getElementById('bgfx')||(()=>{const k=document.createElement('canvas');k.id='bgfx';document.body.appendChild(k);return k})()
  const d=c.getContext('2d',{alpha:true});if(!d)return
  let w=innerWidth,h=innerHeight,pr=devicePixelRatio
  function S(){pr=devicePixelRatio;w=innerWidth;h=innerHeight;c.width=w*pr;c.height=h*pr;d.setTransform(pr,0,0,pr,0,0)}S();addEventListener('resize',S)
  d.lineCap='round'
  let vx=w*.78,vy=h*.5
  const label=new URLSearchParams(location.search).get('node')||'观点'
  const N=1400,P=Array.from({length:N},()=>({x:Math.random()*w*.28,y:Math.random()*h,s:1+Math.random()*3,h:200+Math.random()*60}))
  const O=Array.from({length:800},()=>({x:vx+(Math.random()*20-10),y:vy+(Math.random()*20-10),a:Math.random()*6.283,s:.8+Math.random()*1.6,h:200+Math.random()*60}))
  function loop(){if(document.visibilityState!=='visible'){requestAnimationFrame(loop);return}
    d.globalCompositeOperation='source-over'
    d.fillStyle='rgba(11,13,18,.12)';d.fillRect(0,0,w,h)
    d.fillStyle='rgba(120,170,255,.08)';d.beginPath();d.arc(vx,vy,70,0,6.283);d.fill();d.strokeStyle='rgba(120,170,255,.18)';d.lineWidth=2;d.beginPath();d.arc(vx,vy,72,0,6.283);d.stroke();d.font='700 22px ui-sans-serif,system-ui';d.textAlign='center';d.textBaseline='middle';d.fillStyle='rgba(180,205,255,.9)';d.shadowColor='rgba(120,170,255,.6)';d.shadowBlur=12;d.fillText(label,vx,vy);d.shadowBlur=0
    d.globalCompositeOperation='lighter'
    for(const q of P){let dx=vx-q.x,dy=vy-q.y,l=Math.hypot(dx,dy)||1;let vn=q.s*(1+2*(1-l/Math.hypot(w,h)));let ox=q.x,oy=q.y;q.x+=dx/l*vn;q.y+=dy/l*vn;let Ln=l/Math.hypot(w,h);let a=.10+.25*Ln;d.strokeStyle=`hsla(${q.h},90%,65%,${a})`;d.lineWidth=1+vn*.07;d.beginPath();d.moveTo(ox,oy);d.lineTo(q.x,q.y);d.stroke();if(q.x>vx-6||q.y<0||q.y>h){q.x=Math.random()*w*.26;q.y=Math.random()*h;q.s=1+Math.random()*3}}
    for(const r of O){let ox=r.x,oy=r.y;r.x+=Math.cos(r.a)*r.s*2.2;r.y+=Math.sin(r.a)*r.s*2.2;r.a+=(Math.random()-.5)*.06;let Ln=Math.hypot(r.x-vx,r.y-vy)/Math.hypot(w,h);let a=.08+.22*Ln;d.strokeStyle=`hsla(${r.h},90%,65%,${a})`;d.lineWidth=1;d.beginPath();d.moveTo(ox,oy);d.lineTo(r.x,r.y);d.stroke();if(r.x<0||r.x>w||r.y<0||r.y>h){r.x=vx+(Math.random()*20-10);r.y=vy+(Math.random()*20-10);r.a=Math.random()*6.283;r.s=.8+Math.random()*1.6}}
    requestAnimationFrame(loop)
  }
  loop()
})()