package main

// ── HTML ──────────────────────────────────────────────────────────────────────

const settingsHTML = `<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><title>SnapFlow Settings</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:"Segoe UI",system-ui,sans-serif;font-size:13px;
     background:#f0f0f0;color:#1c1c1e;height:100vh;display:flex;flex-direction:column}
.toolbar{background:#ececec;border-bottom:1px solid #c8c8c8;padding:10px 0 0;
         display:flex;flex-direction:column;align-items:stretch}
.toolbar-inner{display:flex;justify-content:space-between;align-items:flex-end;padding:0 14px}
.toolbar-tabs{display:flex}
.tab-btn{border:none;background:transparent;padding:6px 0 8px;width:90px;font-size:11px;
         color:#555;cursor:pointer;outline:none;font-family:inherit;
         display:flex;flex-direction:column;align-items:center;gap:3px;
         border-bottom:2.5px solid transparent;transition:color .15s}
.tab-btn.active{color:#1a1a1a;border-bottom-color:#1a1a1a}
.tab-btn svg{opacity:.55}.tab-btn.active svg{opacity:1}
.lang-group{display:flex;gap:3px;padding-bottom:8px}
.lang-btn{border:1px solid #c8c8c8;background:white;padding:2px 10px;font-size:11px;
          font-family:inherit;cursor:pointer;border-radius:4px;color:#666;outline:none}
.lang-btn.active{background:#1a1a1a;color:white;border-color:#1a1a1a}
.page{display:none;flex:1;overflow:hidden;flex-direction:column;background:white}
.page.active{display:flex}
/* Shortcuts */
.list-wrap{flex:1;overflow-y:auto}
.sec-hdr{padding:5px 12px 3px;font-size:10px;font-weight:700;color:#888;
         text-transform:uppercase;letter-spacing:.6px;background:#f5f5f5;
         border-bottom:1px solid #e8e8e8;position:sticky;top:0;z-index:1}
.sc-row{display:flex;align-items:center;padding:0 12px;height:32px;
        border-bottom:1px solid #f0f0f0;gap:8px}
.sc-row:hover{background:#f8f9ff}
.sc-diagram{flex-shrink:0;width:36px;height:24px;display:flex;align-items:center}
.sc-name{flex:1;font-size:12.5px}
.chip{min-width:126px;height:22px;display:flex;align-items:center;justify-content:center;
      border-radius:5px;font-size:11.5px;padding:0 9px;cursor:pointer;user-select:none;
      font-family:Consolas,monospace;transition:all .12s}
.chip.on{background:#eef2ff;border:1px solid #c7d2fe;color:#3730a3}
.chip.on:hover{background:#e0e7ff}
.chip.off{background:#f4f4f5;border:1px solid #e0e0e0;color:#aaa;font-family:inherit;font-size:12px}
.chip.rec{background:#fffbeb;border:1.5px solid #f59e0b;color:#92400e;
          animation:blink .9s step-end infinite}
@keyframes blink{50%{border-color:transparent}}
/* Snap Areas */
.snap-page{flex:1;overflow-y:auto;background:#f5f5f5;padding:12px 16px}
.winsnap-row{margin-bottom:12px;padding:9px 14px;background:white;border:1px solid #e0e0e0;
             border-radius:8px;display:flex;align-items:center;gap:10px}
.winsnap-lbl{flex:1}
.winsnap-lbl-main{font-size:13px;color:#1a1a1a;font-weight:600}
.winsnap-lbl-sub{font-size:11px;color:#888;margin-top:1px}
.snap-opts{display:flex;flex-wrap:wrap;gap:8px 28px;margin-bottom:12px}
.snap-opts label{display:flex;align-items:center;gap:6px;font-size:12.5px;cursor:pointer}
.snap-opts input[type=checkbox]{width:14px;height:14px;accent-color:#0078d4}
.thirds-info{display:none;margin-bottom:12px;padding:10px 14px;
             background:#eff6ff;border:1px solid #bfdbfe;border-radius:8px;font-size:12px;color:#1e40af}
.thirds-info-title{font-weight:700;margin-bottom:6px}
.thirds-info-body{display:flex;gap:12px;align-items:center}
.thirds-info-text{line-height:1.6}
/* Zone grid */
.snap-grid{display:grid;grid-template-columns:1fr 2fr 1fr;gap:7px;
           max-width:570px;margin:0 auto}
.zone-cell{background:white;border:1px solid #ddd;border-radius:7px;
           padding:5px 6px 6px;display:flex;flex-direction:column;align-items:center;gap:3px}
.zone-cell-lbl{font-size:9px;font-weight:700;color:#999;text-transform:uppercase;
               letter-spacing:.4px;align-self:flex-start}
.zone-cell-diag{display:flex;align-items:center;justify-content:center}
.zone-sel{background:white;border:1px solid #c8c8c8;border-radius:5px;
          padding:2px 4px;font-size:10.5px;font-family:inherit;color:#1a1a1a;
          cursor:pointer;outline:none;width:100%;appearance:auto;height:26px}
.zone-sel:focus{border-color:#0078d4;box-shadow:0 0 0 2px rgba(0,120,212,.2)}
.screen-cell{background:linear-gradient(135deg,#1a1a2e 0%,#16213e 50%,#0f3460 100%);
             border-radius:8px;border:2px solid #555;display:flex;align-items:center;
             justify-content:center;position:relative;overflow:hidden;min-height:110px}
.screen-cell::before{content:'';position:absolute;inset:0;
  background:radial-gradient(ellipse at 30% 40%,rgba(0,120,212,.3) 0%,transparent 60%)}
.screen-icon{color:rgba(255,255,255,.4);font-size:26px;z-index:1}
/* General */
.gen-wrap{flex:1;overflow-y:auto;padding:16px 20px;background:#f5f5f5}
.gen-card{background:white;border-radius:10px;border:1px solid #e0e0e0;margin-bottom:14px;overflow:hidden}
.gen-card-title{padding:10px 16px;font-size:11px;font-weight:700;color:#666;
                text-transform:uppercase;letter-spacing:.5px;
                border-bottom:1px solid #f0f0f0;background:#fafafa}
.gen-row{display:flex;align-items:center;padding:9px 16px;gap:10px;border-bottom:1px solid #f8f8f8}
.gen-row:last-child{border-bottom:none}
.gen-lbl{flex:1}
.gen-lbl-main{font-size:13px;color:#1a1a1a}
.gen-lbl-sub{font-size:11px;color:#888;margin-top:1px}
.num-in{width:68px;height:26px;border:1px solid #c8c8c8;border-radius:6px;padding:0 8px;
        font-size:13px;font-family:inherit;text-align:center;background:#fafafa;outline:none}
.num-in:focus{border-color:#0078d4;background:white;box-shadow:0 0 0 2px rgba(0,120,212,.15)}
.gen-sfx{font-size:11.5px;color:#888;min-width:20px}
/* Footer */
.footer{display:flex;align-items:center;padding:9px 14px;
        background:#ececec;border-top:1px solid #c8c8c8;gap:8px}
.fl{flex:1;display:flex;gap:8px}
.btn{height:26px;padding:0 13px;border-radius:5px;font-size:12px;font-family:inherit;
     cursor:pointer;border:1px solid #bbb;background:white;color:#333;outline:none}
.btn:hover{background:#e8e8e8}
.btn.ok{background:#0078d4;color:white;border-color:#006abc}
.btn.ok:hover{background:#006abc}
.saved{font-size:11.5px;color:#107c10;font-weight:500;opacity:0;transition:opacity .3s}
.saved.show{opacity:1}
</style></head>
<body>
<div class="toolbar">
  <div class="toolbar-inner">
    <div class="toolbar-tabs">
      <button class="tab-btn active" id="btn-shortcuts" onclick="showTab('shortcuts',this)">
        <svg width="22" height="22" viewBox="0 0 22 22" fill="none">
          <rect x="2" y="5" width="18" height="12" rx="2" stroke="currentColor" stroke-width="1.5" fill="none"/>
          <text x="11" y="14" text-anchor="middle" font-size="6" fill="currentColor">KB</text>
        </svg>
        <span data-i18n="tabShortcuts">Shortcuts</span>
      </button>
      <button class="tab-btn" id="btn-snapAreas" onclick="showTab('snapAreas',this)">
        <svg width="22" height="22" viewBox="0 0 22 22" fill="none">
          <rect x="2" y="2" width="8" height="8" rx="1.5" stroke="currentColor" stroke-width="1.5" fill="none"/>
          <rect x="12" y="2" width="8" height="8" rx="1.5" stroke="currentColor" stroke-width="1.5" fill="none"/>
          <rect x="2" y="12" width="8" height="8" rx="1.5" stroke="currentColor" stroke-width="1.5" fill="none"/>
          <rect x="12" y="12" width="8" height="8" rx="1.5" stroke="currentColor" stroke-width="1.5" fill="none"/>
        </svg>
        <span data-i18n="tabSnapAreas">Snap Areas</span>
      </button>
      <button class="tab-btn" id="btn-general" onclick="showTab('general',this)">
        <svg width="22" height="22" viewBox="0 0 22 22" fill="none">
          <circle cx="11" cy="11" r="3" stroke="currentColor" stroke-width="1.5" fill="none"/>
          <path d="M11 2v2M11 18v2M2 11h2M18 11h2M4.9 4.9l1.4 1.4M15.7 15.7l1.4 1.4M4.9 17.1l1.4-1.4M15.7 6.3l1.4-1.4"
                stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/>
        </svg>
        <span data-i18n="tabGeneral">General</span>
      </button>
    </div>
    <div class="lang-group">
      <button class="lang-btn active" data-lang="en" onclick="setLang('en')">EN</button>
      <button class="lang-btn" data-lang="es" onclick="setLang('es')">ES</button>
    </div>
  </div>
</div>

<div id="page-shortcuts" class="page active">
  <div class="list-wrap"><div id="scList"></div></div>
</div>

<div id="page-snapAreas" class="page">
  <div class="snap-page">
    <div class="winsnap-row">
      <div class="winsnap-lbl">
        <div class="winsnap-lbl-main" data-i18n="winSnapTitle">Windows Snap (AeroSnap)</div>
        <div class="winsnap-lbl-sub" data-i18n="winSnapSub">Disable to prevent Windows from intercepting drag-to-edge gestures</div>
      </div>
      <label style="display:flex;align-items:center;gap:6px;font-size:12px;cursor:pointer">
        <input type="checkbox" id="winSnapEnabled" style="width:14px;height:14px;accent-color:#0078d4"
               onchange="goSetWindowsSnap(this.checked)">
        <span data-i18n="winSnapEnabled">Enabled</span>
      </label>
    </div>
    <div class="snap-opts">
      <label><input type="checkbox" id="snapByDragging"> <span data-i18n="cbSnap">Snap windows by dragging</span></label>
      <label><input type="checkbox" id="restoreSize"> <span data-i18n="cbRestore">Restore window size when unsnapped</span></label>
      <label><input type="checkbox" id="animateFootprint"> <span data-i18n="cbAnimate">Animate footprint</span></label>
    </div>
    <div class="thirds-info" id="thirdsInfo">
      <div class="thirds-info-title" data-i18n="thirdsTitle">Bottom edge — Compound Thirds</div>
      <div class="thirds-info-body">
        <svg width="114" height="44" viewBox="0 0 114 44" style="flex-shrink:0">
          <rect x="2" y="2" width="110" height="40" rx="3" fill="#dbeafe" stroke="#93c5fd" stroke-width="1.2"/>
          <line x1="38.7" y1="2" x2="38.7" y2="42" stroke="#93c5fd" stroke-width="0.8" stroke-dasharray="2,2"/>
          <line x1="75.3" y1="2" x2="75.3" y2="42" stroke="#93c5fd" stroke-width="0.8" stroke-dasharray="2,2"/>
          <rect x="2" y="30" width="110" height="12" fill="#93c5fd" opacity="0.5" rx="0"/>
          <rect x="3" y="31" width="34.7" height="10" fill="#3b82f6" opacity="0.5" rx="1"/>
          <rect x="40.7" y="31" width="33.6" height="10" fill="#1d4ed8" opacity="0.85" rx="1"/>
          <rect x="76.3" y="31" width="34.7" height="10" fill="#3b82f6" opacity="0.5" rx="1"/>
          <text x="20" y="22" text-anchor="middle" font-size="9" fill="#1d4ed8">1/3</text>
          <text x="57" y="22" text-anchor="middle" font-size="9" fill="#1d4ed8">1/3</text>
          <text x="94" y="22" text-anchor="middle" font-size="9" fill="#1d4ed8">1/3</text>
        </svg>
        <div class="thirds-info-text">
          <div data-i18n="thirdsP">Drag a window to the bottom edge and release:</div>
          <div>• <span data-i18n="thirdsL1">In the <b>first third</b> → snaps to left 1/3</span></div>
          <div>• <span data-i18n="thirdsL2"><b>Drag across two thirds</b> → snaps to 2/3</span></div>
          <div>• <span data-i18n="thirdsL3"><b>All three thirds</b> → maximizes</span></div>
        </div>
      </div>
    </div>
    <div class="snap-grid" id="snapGrid"></div>
  </div>
</div>

<div id="page-general" class="page">
  <div class="gen-wrap" id="genWrap"></div>
</div>

<div class="footer">
  <div class="fl">
    <button class="btn" onclick="resetDefaults()"><span data-i18n="btnReset">Reset defaults</span></button>
    <button class="btn" onclick="goOpenConfigFile()"><span data-i18n="btnConfig">Open config file</span></button>
  </div>
  <span class="saved" id="savedMsg"><span data-i18n="saved">Saved ✓</span></span>
  <button class="btn" onclick="window.close?window.close():history.back()"><span data-i18n="btnCancel">Cancel</span></button>
  <button class="btn ok" onclick="save()"><span data-i18n="btnSave">Save</span></button>
</div>

<script>
// ── i18n ──────────────────────────────────────────────────────────────────────
const LANGS = {
  en:{
    tabShortcuts:'Shortcuts',tabSnapAreas:'Snap Areas',tabGeneral:'General',
    winSnapTitle:'Windows Snap (AeroSnap)',
    winSnapSub:'Disable to prevent Windows from intercepting drag-to-edge gestures',
    winSnapEnabled:'Enabled',
    cbSnap:'Snap windows by dragging',cbRestore:'Restore window size when unsnapped',cbAnimate:'Animate footprint',
    zTL:'Top-Left',zT:'Top',zTR:'Top-Right',zL:'Left',zR:'Right',zBL:'Bot-Left',zB:'Bottom',zBR:'Bot-Right',
    thirdsTitle:'Bottom edge — Compound Thirds',
    thirdsP:'Drag a window to the bottom edge and release:',
    thirdsL1:'In the <b>first third</b> → snaps to left 1/3',
    thirdsL2:'<b>Drag across two thirds</b> → snaps to 2/3',
    thirdsL3:'<b>All three thirds</b> → maximizes',
    gSnapZones:'Snap Zones',gEdge:'Edge threshold',gEdgeSub:'Distance from screen edge to activate snap',
    gCorner:'Corner hot zone',gCornerSub:'Size of corner snap squares',
    gShort:'Short edge zone',gShortSub:'Top/bottom band height on left/right edges',
    gGap:'Gap Between Windows',gGapSize:'Gap size',gGapSizeSub:'Space inserted between snapped windows',
    gAlmost:'Almost Maximize',gW:'Width',gH:'Height',
    gResize:'Resize Step',gStep:'Step size (Larger / Smaller actions)',
    btnReset:'Reset defaults',btnConfig:'Open config file',btnCancel:'Cancel',btnSave:'Save',saved:'Saved ✓',
  },
  es:{
    tabShortcuts:'Atajos',tabSnapAreas:'Zonas',tabGeneral:'General',
    winSnapTitle:'Windows Snap (AeroSnap)',
    winSnapSub:'Desactivar para evitar que Windows intercepte gestos de arrastre al borde',
    winSnapEnabled:'Activado',
    cbSnap:'Ajustar ventanas arrastrando',cbRestore:'Restaurar tamaño al desanclar',cbAnimate:'Animar huella',
    zTL:'Sup-Izq',zT:'Arriba',zTR:'Sup-Der',zL:'Izquierda',zR:'Derecha',zBL:'Inf-Izq',zB:'Abajo',zBR:'Inf-Der',
    thirdsTitle:'Borde inferior — Tercios compuestos',
    thirdsP:'Arrastra una ventana al borde inferior y suéltala:',
    thirdsL1:'En el <b>primer tercio</b> → ocupa 1/3 izquierdo',
    thirdsL2:'<b>Arrastra entre dos tercios</b> → ocupa 2/3',
    thirdsL3:'<b>Los tres tercios</b> → maximiza',
    gSnapZones:'Zonas de ajuste',gEdge:'Umbral de borde',gEdgeSub:'Distancia desde el borde de pantalla para activar el ajuste',
    gCorner:'Zona de esquina',gCornerSub:'Tamaño de los cuadrados de esquina',
    gShort:'Zona de borde corto',gShortSub:'Altura de la banda superior/inferior en bordes izq/der',
    gGap:'Espacio entre ventanas',gGapSize:'Tamaño del espacio',gGapSizeSub:'Espacio entre ventanas ajustadas',
    gAlmost:'Casi maximizar',gW:'Ancho',gH:'Alto',
    gResize:'Paso de redimensión',gStep:'Tamaño de paso (Ampliar / Reducir)',
    btnReset:'Restaurar valores',btnConfig:'Abrir configuración',btnCancel:'Cancelar',btnSave:'Guardar',saved:'Guardado ✓',
  },
};
let curLang = 'en';
function T(k){ return LANGS[curLang][k] || LANGS.en[k] || k; }

function setLang(l){
  curLang = l;
  document.querySelectorAll('[data-i18n]').forEach(el => { el.innerHTML = T(el.dataset.i18n); });
  document.querySelectorAll('.lang-btn').forEach(b => b.classList.toggle('active', b.dataset.lang === l));
  buildGeneral();
}

// ── Tab switching ─────────────────────────────────────────────────────────────
function showTab(id, btn) {
  document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));
  document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
  document.getElementById('page-' + id).classList.add('active');
  btn.classList.add('active');
}

// ── Action diagrams ───────────────────────────────────────────────────────────
const ACT = {
  0:[0,0,.5,1],1:[.5,0,.5,1],10:[0,.5,1,.5],11:[0,0,1,.5],30:[.25,0,.5,1],
  13:[0,.5,.5,.5],14:[.5,.5,.5,.5],15:[0,0,.5,.5],16:[.5,0,.5,.5],
  20:[0,0,.333,1],21:[0,0,.667,1],22:[.333,0,.333,1],23:[.333,0,.667,1],
  24:[.667,0,.333,1],84:[.167,0,.667,1],
  87:[0,0,1,.333],88:[0,.333,1,.333],89:[0,.667,1,.333],
  90:[0,0,1,.667],91:[0,.333,1,.667],
  31:[0,0,.25,1],32:[.25,0,.25,1],33:[.5,0,.25,1],34:[.75,0,.25,1],
  35:[0,0,.75,1],85:[.125,0,.75,1],36:[.25,0,.75,1],
  37:[0,0,.333,.5],38:[.333,0,.333,.5],39:[.667,0,.333,.5],
  40:[0,.5,.333,.5],41:[.333,.5,.333,.5],42:[.667,.5,.333,.5],
  58:[0,0,.25,.5],59:[.25,0,.25,.5],60:[.5,0,.25,.5],61:[.75,0,.25,.5],
  62:[0,.5,.25,.5],63:[.25,.5,.25,.5],64:[.5,.5,.25,.5],65:[.75,.5,.25,.5],
  45:[0,0,.333,.333],46:[.333,0,.333,.333],47:[.667,0,.333,.333],
  48:[0,.333,.333,.333],49:[.333,.333,.333,.333],50:[.667,.333,.333,.333],
  51:[0,.667,.333,.333],52:[.333,.667,.333,.333],53:[.667,.667,.333,.333],
  54:[0,0,.333,.5],55:[.667,0,.333,.5],56:[0,.5,.333,.5],57:[.667,.5,.333,.5],
  92:[0,0,.25,.333],93:[.25,0,.25,.333],94:[.5,0,.25,.333],95:[.75,0,.25,.333],
  96:[0,.333,.25,.333],97:[.25,.333,.25,.333],98:[.5,.333,.25,.333],99:[.75,.333,.25,.333],
  100:[0,.667,.25,.333],101:[.25,.667,.25,.333],102:[.5,.667,.25,.333],103:[.75,.667,.25,.333],
  104:[0,0,.25,.25],105:[.25,0,.25,.25],106:[.5,0,.25,.25],107:[.75,0,.25,.25],
  108:[0,.25,.25,.25],109:[.25,.25,.25,.25],110:[.5,.25,.25,.25],111:[.75,.25,.25,.25],
  112:[0,.5,.25,.25],113:[.25,.5,.25,.25],114:[.5,.5,.25,.25],115:[.75,.5,.25,.25],
  116:[0,.75,.25,.25],117:[.25,.75,.25,.25],118:[.5,.75,.25,.25],119:[.75,.75,.25,.25],
  2:[0,0,1,1],29:[.05,.05,.9,.9],12:[.25,.25,.5,.5],71:[.125,.125,.75,.75],
};

// zone string value → action int for diagrams
const ZONE_INT = {
  compoundThirds:-1,none:-999,
  leftHalf:0,rightHalf:1,topHalf:11,bottomHalf:10,centerHalf:30,
  topLeft:15,topRight:16,bottomLeft:13,bottomRight:14,
  firstThird:20,centerThird:22,lastThird:24,
  firstTwoThirds:21,centerTwoThirds:84,lastTwoThirds:23,
  topVerticalThird:87,middleVerticalThird:88,bottomVerticalThird:89,
  topVerticalTwoThirds:90,bottomVerticalTwoThirds:91,
  firstFourth:31,secondFourth:32,thirdFourth:33,lastFourth:34,
  firstThreeFourths:35,centerThreeFourths:85,lastThreeFourths:36,
  topLeftSixth:37,topCenterSixth:38,topRightSixth:39,
  bottomLeftSixth:40,bottomCenterSixth:41,bottomRightSixth:42,
  topLeftThird:54,topRightThird:55,bottomLeftThird:56,bottomRightThird:57,
  maximize:2,almostMaximize:29,maximizeHeight:3,center:12,centerProminently:71,
};

const DW=36,DH=24,DP=1.5;
function makeDiagram(aid){
  const bg='#e8eaed',hl='#0078d4',br='#b0b8c4';
  const iw=DW-2*DP,ih=DH-2*DP;
  const screen=` + "`" + `<rect x="${DP}" y="${DP}" width="${iw}" height="${ih}" rx="2" fill="${bg}" stroke="${br}" stroke-width="0.8"/>` + "`" + `;
  if(aid===-1){
    // compound thirds: three thirds shown in bottom strip
    const bh=ih*.38,by=DP+ih-bh,tw=(iw-2)/3;
    return ` + "`" + `<svg width="${DW}" height="${DH}" viewBox="0 0 ${DW} ${DH}">${screen}
      <rect x="${DP}"          y="${by}" width="${tw-.5}" height="${bh}" fill="${hl}" rx="1" opacity=".4"/>
      <rect x="${DP+tw+.5}"    y="${by}" width="${tw-1}"  height="${bh}" fill="${hl}" rx="1" opacity=".9"/>
      <rect x="${DP+tw*2+.5}"  y="${by}" width="${tw-.5}" height="${bh}" fill="${hl}" rx="1" opacity=".4"/>
    </svg>` + "`" + `;
  }
  if(aid===3){
    return ` + "`" + `<svg width="${DW}" height="${DH}" viewBox="0 0 ${DW} ${DH}">${screen}
      <rect x="${DP+iw*.3}" y="${DP}" width="${iw*.4}" height="${ih}" fill="${hl}" rx="1"/></svg>` + "`" + `;
  }
  const c=ACT[aid];
  if(!c) return ` + "`" + `<svg width="${DW}" height="${DH}" viewBox="0 0 ${DW} ${DH}">${screen}</svg>` + "`" + `;
  const [lf,tf,wf,hf]=c;
  return ` + "`" + `<svg width="${DW}" height="${DH}" viewBox="0 0 ${DW} ${DH}">${screen}
    <rect x="${DP+lf*iw}" y="${DP+tf*ih}" width="${wf*iw}" height="${hf*ih}" fill="${hl}" rx="1"/></svg>` + "`" + `;
}

// ── Shortcuts ─────────────────────────────────────────────────────────────────
let recordingChip=null;
function buildShortcuts(){
  const list=document.getElementById('scList');
  let sec='';
  window._shortcuts.forEach((e,i)=>{
    if(e.section!==sec){
      sec=e.section;
      list.insertAdjacentHTML('beforeend','<div class="sec-hdr">'+e.section+'</div>');
    }
    const on=!!e.text;
    list.insertAdjacentHTML('beforeend',` + "`" + `
      <div class="sc-row">
        <span class="sc-diagram">${makeDiagram(e.action)}</span>
        <span class="sc-name">${e.label}</span>
        <div class="chip ${on?'on':'off'}" data-i="${i}" onclick="toggleRec(this,${i})">${e.text||'—'}</div>
      </div>` + "`" + `);
  });
}
function toggleRec(chip,idx){
  if(recordingChip){
    const ri=parseInt(recordingChip.dataset.i);
    recordingChip.className='chip '+(window._shortcuts[ri].text?'on':'off');
    recordingChip.textContent=window._shortcuts[ri].text||'—';
    if(recordingChip===chip){recordingChip=null;return;}
  }
  recordingChip=chip;chip.className='chip rec';chip.textContent='Press keys…';
}
document.addEventListener('keydown',e=>{
  if(!recordingChip)return;
  e.preventDefault();
  const idx=parseInt(recordingChip.dataset.i);
  if(e.key==='Escape'){
    recordingChip.className='chip '+(window._shortcuts[idx].text?'on':'off');
    recordingChip.textContent=window._shortcuts[idx].text||'—';
    recordingChip=null;return;
  }
  if(['Control','Alt','Shift','Meta','CapsLock'].includes(e.key))return;
  let s='';
  if(e.ctrlKey)s+='Ctrl+';if(e.altKey)s+='Alt+';
  if(e.shiftKey)s+='Shift+';if(e.metaKey)s+='Win+';
  const km={'ArrowLeft':'←','ArrowRight':'→','ArrowUp':'↑','ArrowDown':'↓',
            'Enter':'↩','Backspace':'⌫','Delete':'⌦','Equal':'=','Minus':'-'};
  s+=km[e.key]||(e.key.length===1?e.key.toUpperCase():e.key);
  window._shortcuts[idx].text=s;
  recordingChip.className='chip on';recordingChip.textContent=s;recordingChip=null;
});

// ── Snap Areas ────────────────────────────────────────────────────────────────
const ALL_ZONE_OPTS=[
  ['leftHalf','Left Half','Halves'],['rightHalf','Right Half','Halves'],
  ['topHalf','Top Half','Halves'],['bottomHalf','Bottom Half','Halves'],['centerHalf','Center Half','Halves'],
  ['topLeft','Top Left','Quarters'],['topRight','Top Right','Quarters'],
  ['bottomLeft','Bottom Left','Quarters'],['bottomRight','Bottom Right','Quarters'],
  ['firstThird','First Third','Thirds'],['centerThird','Center Third','Thirds'],['lastThird','Last Third','Thirds'],
  ['firstTwoThirds','First Two Thirds','Thirds'],['centerTwoThirds','Center Two Thirds','Thirds'],
  ['lastTwoThirds','Last Two Thirds','Thirds'],
  ['topVerticalThird','Top Third','Vertical Thirds'],['middleVerticalThird','Middle Third','Vertical Thirds'],
  ['bottomVerticalThird','Bottom Third','Vertical Thirds'],
  ['topVerticalTwoThirds','Top Two Thirds','Vertical Thirds'],['bottomVerticalTwoThirds','Bottom Two Thirds','Vertical Thirds'],
  ['firstFourth','First Fourth','Fourths'],['secondFourth','Second Fourth','Fourths'],
  ['thirdFourth','Third Fourth','Fourths'],['lastFourth','Last Fourth','Fourths'],
  ['firstThreeFourths','First Three Fourths','Fourths'],['centerThreeFourths','Center Three Fourths','Fourths'],
  ['lastThreeFourths','Last Three Fourths','Fourths'],
  ['topLeftSixth','Top Left Sixth','Sixths'],['topCenterSixth','Top Center Sixth','Sixths'],
  ['topRightSixth','Top Right Sixth','Sixths'],['bottomLeftSixth','Bottom Left Sixth','Sixths'],
  ['bottomCenterSixth','Bottom Center Sixth','Sixths'],['bottomRightSixth','Bottom Right Sixth','Sixths'],
  ['topLeftThird','Top Left Third','Corner Thirds'],['topRightThird','Top Right Third','Corner Thirds'],
  ['bottomLeftThird','Bottom Left Third','Corner Thirds'],['bottomRightThird','Bottom Right Third','Corner Thirds'],
  ['maximize','Maximize','Maximize'],['almostMaximize','Almost Maximize','Maximize'],
  ['maximizeHeight','Maximize Height','Maximize'],
  ['center','Center','Center'],['centerProminently','Center Prominently','Center'],
  ['none','None',''],
];

const botOpts=[
  ['compoundThirds','Thirds — drag to span  (recommended)','Compound'],
  ...ALL_ZONE_OPTS,
];
const topAllowed=new Set(['maximize','almostMaximize','maximizeHeight','topHalf',
  'topVerticalThird','topVerticalTwoThirds','center','centerProminently',
  'firstThird','centerThird','lastThird','none']);
const topOpts=ALL_ZONE_OPTS.filter(([v])=>topAllowed.has(v));
const leftAllowed=new Set(['leftHalf','firstThird','firstTwoThirds','firstFourth',
  'firstThreeFourths','topLeft','bottomLeft','topLeftSixth','bottomLeftSixth',
  'topLeftThird','bottomLeftThird','none']);
const leftOpts=[['leftHalf','Left Half  (+ subdivided edge)','Halves'],
  ...ALL_ZONE_OPTS.filter(([v])=>leftAllowed.has(v)&&v!=='leftHalf')];
const rightAllowed=new Set(['rightHalf','lastThird','lastTwoThirds','lastFourth',
  'lastThreeFourths','topRight','bottomRight','topRightSixth','bottomRightSixth',
  'topRightThird','bottomRightThird','none']);
const rightOpts=[['rightHalf','Right Half  (+ subdivided edge)','Halves'],
  ...ALL_ZONE_OPTS.filter(([v])=>rightAllowed.has(v)&&v!=='rightHalf')];
const cornerOpts=ALL_ZONE_OPTS;

function mkSelHtml(id,opts,val){
  let h='<select class="zone-sel" id="'+id+'" onchange="updateZoneDiag(\''+id+'\')">',g='';
  opts.forEach(([v,l,grp])=>{
    if(grp&&grp!==g&&grp!==''){ if(g)h+='</optgroup>'; h+='<optgroup label="'+grp+'">'; g=grp; }
    h+='<option value="'+v+'"'+(v===val?' selected':'')+'>'+l+'</option>';
  });
  if(g)h+='</optgroup>';
  return h+'</select>';
}

function mkZoneCell(id,labelKey,opts,val){
  const aid=ZONE_INT[val]??-999;
  return ` + "`" + `<div class="zone-cell">
    <div class="zone-cell-lbl">${T(labelKey)}</div>
    <div class="zone-cell-diag" id="diag-${id}">${makeDiagram(aid)}</div>
    ${mkSelHtml('z-'+id,opts,val)}
  </div>` + "`" + `;
}

function updateZoneDiag(selId){
  const id=selId.replace(/^z-/,'');
  const sel=document.getElementById(selId);
  const diag=document.getElementById('diag-'+id);
  if(sel&&diag){ diag.innerHTML=makeDiagram(ZONE_INT[sel.value]??-999); }
  if(id==='bottom') updateThirdsInfo();
}

function buildSnapAreas(){
  const sa=window._snapAreas;
  document.getElementById('snapByDragging').checked=sa.snapByDragging;
  document.getElementById('restoreSize').checked=sa.restoreSize;
  document.getElementById('animateFootprint').checked=sa.animateFootprint;
  document.getElementById('winSnapEnabled').checked=window._winSnap;
  const g=document.getElementById('snapGrid');
  g.innerHTML=
    mkZoneCell('topLeft','zTL',cornerOpts,sa.topLeft)+
    mkZoneCell('top','zT',topOpts,sa.top)+
    mkZoneCell('topRight','zTR',cornerOpts,sa.topRight)+
    mkZoneCell('left','zL',leftOpts,sa.left)+
    '<div class="screen-cell"><span class="screen-icon">🖥</span></div>'+
    mkZoneCell('right','zR',rightOpts,sa.right)+
    mkZoneCell('bottomLeft','zBL',cornerOpts,sa.bottomLeft)+
    mkZoneCell('bottom','zB',botOpts,sa.bottom)+
    mkZoneCell('bottomRight','zBR',cornerOpts,sa.bottomRight);
  updateThirdsInfo();
  document.getElementById('z-bottom').addEventListener('change',updateThirdsInfo);
}

function updateThirdsInfo(){
  const sel=document.getElementById('z-bottom');
  document.getElementById('thirdsInfo').style.display=
    sel&&sel.value==='compoundThirds'?'block':'none';
}

// ── General ───────────────────────────────────────────────────────────────────
function buildGeneral(){
  const g=window._general;
  document.getElementById('genWrap').innerHTML=` + "`" + `
    <div class="gen-card">
      <div class="gen-card-title">${T('gSnapZones')}</div>
      <div class="gen-row">
        <div class="gen-lbl"><div class="gen-lbl-main">${T('gEdge')}</div>
          <div class="gen-lbl-sub">${T('gEdgeSub')}</div></div>
        <input class="num-in" id="edgeThreshold" type="number" min="1" max="200" value="${g.edgeThreshold}">
        <span class="gen-sfx">px</span>
      </div>
      <div class="gen-row">
        <div class="gen-lbl"><div class="gen-lbl-main">${T('gCorner')}</div>
          <div class="gen-lbl-sub">${T('gCornerSub')}</div></div>
        <input class="num-in" id="cornerSize" type="number" min="1" max="200" value="${g.cornerSize}">
        <span class="gen-sfx">px</span>
      </div>
      <div class="gen-row">
        <div class="gen-lbl"><div class="gen-lbl-main">${T('gShort')}</div>
          <div class="gen-lbl-sub">${T('gShortSub')}</div></div>
        <input class="num-in" id="shortEdgeSize" type="number" min="1" max="500" value="${g.shortEdgeSize}">
        <span class="gen-sfx">px</span>
      </div>
    </div>
    <div class="gen-card">
      <div class="gen-card-title">${T('gGap')}</div>
      <div class="gen-row">
        <div class="gen-lbl"><div class="gen-lbl-main">${T('gGapSize')}</div>
          <div class="gen-lbl-sub">${T('gGapSizeSub')}</div></div>
        <input class="num-in" id="gapSize" type="number" min="0" max="100" value="${g.gapSize}">
        <span class="gen-sfx">px</span>
      </div>
    </div>
    <div class="gen-card">
      <div class="gen-card-title">${T('gAlmost')}</div>
      <div class="gen-row">
        <div class="gen-lbl"><div class="gen-lbl-main">${T('gW')}</div></div>
        <input class="num-in" id="almostMaxWidth" type="number" min="50" max="100" value="${Math.round(g.almostMaxWidth)}">
        <span class="gen-sfx">%</span>
      </div>
      <div class="gen-row">
        <div class="gen-lbl"><div class="gen-lbl-main">${T('gH')}</div></div>
        <input class="num-in" id="almostMaxHeight" type="number" min="50" max="100" value="${Math.round(g.almostMaxHeight)}">
        <span class="gen-sfx">%</span>
      </div>
    </div>
    <div class="gen-card">
      <div class="gen-card-title">${T('gResize')}</div>
      <div class="gen-row">
        <div class="gen-lbl"><div class="gen-lbl-main">${T('gStep')}</div></div>
        <input class="num-in" id="sizeStep" type="number" min="1" max="200" value="${g.sizeStep}">
        <span class="gen-sfx">px</span>
      </div>
    </div>` + "`" + `;
}

// ── Save / Reset ──────────────────────────────────────────────────────────────
function save(){
  const gi=id=>parseInt(document.getElementById(id)?.value||'0',10);
  const gv=id=>document.getElementById(id)?.value||'';
  const gc=id=>document.getElementById(id)?.checked||false;
  goSaveSettings(JSON.stringify({
    general:{edgeThreshold:gi('edgeThreshold'),cornerSize:gi('cornerSize'),
             shortEdgeSize:gi('shortEdgeSize'),gapSize:gi('gapSize'),
             almostMaxWidth:gi('almostMaxWidth'),almostMaxHeight:gi('almostMaxHeight'),
             sizeStep:gi('sizeStep')},
    snapAreas:{snapByDragging:gc('snapByDragging'),restoreSize:gc('restoreSize'),
               animateFootprint:gc('animateFootprint'),
               topLeft:gv('z-topLeft'),top:gv('z-top'),topRight:gv('z-topRight'),
               left:gv('z-left'),right:gv('z-right'),
               bottomLeft:gv('z-bottomLeft'),bottom:gv('z-bottom'),bottomRight:gv('z-bottomRight')}
  }));
}
function resetDefaults(){
  const d={edgeThreshold:46,cornerSize:20,shortEdgeSize:145,
           gapSize:0,almostMaxWidth:90,almostMaxHeight:90,sizeStep:30};
  Object.entries(d).forEach(([k,v])=>{const el=document.getElementById(k);if(el)el.value=v;});
  const z={topLeft:'topLeft',top:'maximize',topRight:'topRight',
           left:'leftHalf',right:'rightHalf',
           bottomLeft:'bottomLeft',bottom:'compoundThirds',bottomRight:'bottomRight'};
  Object.entries(z).forEach(([k,v])=>{
    const sel=document.getElementById('z-'+k);
    if(sel){sel.value=v;updateZoneDiag('z-'+k);}
  });
  document.getElementById('snapByDragging').checked=true;
  document.getElementById('animateFootprint').checked=true;
}

window.showSaved=function(){
  const m=document.getElementById('savedMsg');
  m.classList.add('show');setTimeout(()=>m.classList.remove('show'),2000);
};

buildShortcuts();
buildSnapAreas();
buildGeneral();
</script>
</body></html>`
