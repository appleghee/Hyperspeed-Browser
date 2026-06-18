(function(){
if(window.__mbSPB)return;
window.__mbSPB=true;

var STORAGE_KEY='spb_config';
var DEFAULT_CONFIG={
  enabled:true,smartMode:true,blockAllPopups:false,autoBlockAds:true,
  notificationDuration:8000,notificationPosition:'br',maxNotifications:3,
  whitelist:[],blacklist:[],blockedCount:0,showBlockedBadge:true,
  theme:'dark',fontSize:'small',soundOnBlock:false,logBlockedToConsole:false,
};
function loadConfig(){
  try{
    var raw=localStorage.getItem(STORAGE_KEY);
    if(raw){var p=JSON.parse(raw);if(typeof p==='object'){
      var m={};for(var k in DEFAULT_CONFIG)m[k]=DEFAULT_CONFIG[k];
      for(var k in p)if(k in m)m[k]=p[k];
      return m;
    }}
  }catch(e){}
  var c={};for(var k in DEFAULT_CONFIG)c[k]=DEFAULT_CONFIG[k];return c;
}
function saveConfig(c){try{localStorage.setItem(STORAGE_KEY,JSON.stringify(c))}catch(e){}}

window.__mbSPBConfig=loadConfig();

function getDomain(url){
  try{
    if(!url||url==='about:blank'||url==='')return'';
    if(url.indexOf('javascript:')===0||url.indexOf('data:')===0)return'special';
    return new URL(url,location.href).hostname.toLowerCase();
  }catch(e){
    var m=url.match(/^(?:https?:\/\/)?([^\/:?#]+)/i);
    return m?m[1].toLowerCase():'';
  }
}

var originalWindowOpen=window.open;
var popupQueue=[];
window.__mbSPBQueue=popupQueue;

function isAdsUrl(url){
  if(!url)return false;
  var p=[/doubleclick\.net/i,/googleadservices/i,/googlesyndication/i,/adnxs\.com/i,/adsystem/i,/adserver/i,/advertisement/i,/popads/i,/popunder/i,/tabunder/i,/trafficjunky/i,/click\.php/i,/track\.php/i,/redirect\.php/i,/\/ad\//i,/\/ads\//i,/\/banner\//i,/\/popup\//i,/sponsor/i,/affiliate/i,/campaign/i,/utm_/i,/\.exe$/i,/\.apk$/i,/\.dmg$/i,/\.msi$/i,/download\.php/i,/file\.php/i];
  for(var i=0;i<p.length;i++){if(p[i].test(url))return true}
  return false;
}

function handlePopupRequest(url,target,features){
  var cfg=window.__mbSPBConfig;
  if(!cfg.enabled)return originalWindowOpen.call(window,url,target,features);
  var td=getDomain(url);var cd=location.hostname.toLowerCase();
  if(url.indexOf('data:')===0||url.indexOf('blob:')===0||url.indexOf('javascript:')===0)return originalWindowOpen.call(window,url,target,features);
  if(url==='about:blank'||url==='')return originalWindowOpen.call(window,url,target,features);
  for(var i=0;i<cfg.blacklist.length;i++){if(td.indexOf(cfg.blacklist[i])>=0||cfg.blacklist[i].indexOf(td)>=0){cfg.blockedCount++;saveConfig(cfg);return null}}
  for(var i=0;i<cfg.whitelist.length;i++){if(td.indexOf(cfg.whitelist[i])>=0||cfg.whitelist[i].indexOf(td)>=0)return originalWindowOpen.call(window,url,target,features)}
  if(cfg.smartMode&&td===cd)return originalWindowOpen.call(window,url,target,features);
  if(cfg.autoBlockAds&&isAdsUrl(url)){cfg.blockedCount++;saveConfig(cfg);return null}
  if(cfg.blockAllPopups){cfg.blockedCount++;saveConfig(cfg);return null}
  popupQueue.push({id:Date.now()+'_'+Math.random().toString(36).substr(2,9),url:url,target:target,features:features,targetDomain:td,currentDomain:cd,timestamp:Date.now()});
  return null;
}

window.open=function(url,target,features){return handlePopupRequest(url,target,features)};
window._originalOpen=originalWindowOpen;

document.addEventListener('click',function(e){
  var cfg=window.__mbSPBConfig;if(!cfg.enabled)return;
  var a=e.target.closest('a');if(!a)return;
  var t=a.getAttribute('target');if(t!=='_blank'&&t!=='_new'&&t!=='popup')return;
  var h=a.getAttribute('href');if(!h||h.indexOf('#')===0||h.indexOf('javascript:')===0)return;
  var u=a.href;if(!u)return;
  var td=getDomain(u);var cd=location.hostname.toLowerCase();
  for(var i=0;i<cfg.whitelist.length;i++){if(td.indexOf(cfg.whitelist[i])>=0||cfg.whitelist[i].indexOf(td)>=0)return}
  if(cfg.smartMode&&td===cd)return;
  e.preventDefault();e.stopImmediatePropagation();
  handlePopupRequest(u,t||'_blank','');
},true);
})();
