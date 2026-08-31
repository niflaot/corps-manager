package customerweb

const pageHTML = `<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>Clientes frecuentes · Benny's Motors</title>
  <style>
    :root{color-scheme:dark;--bg:#0b0d10;--panel:#15181d;--line:#2a3038;--text:#f4f6f8;--muted:#9aa3ae;--green:#2ecc71;--blue:#3498db}
    *{box-sizing:border-box}body{margin:0;background:radial-gradient(circle at top,#17202a 0,#0b0d10 38%);color:var(--text);font:15px/1.45 system-ui,sans-serif}
    main{width:min(1120px,calc(100% - 32px));margin:42px auto}header{display:flex;gap:18px;align-items:center;margin-bottom:24px}
    .logo{display:grid;place-items:center;width:58px;height:58px;border-radius:16px;background:linear-gradient(135deg,var(--green),var(--blue));font-size:28px}
    h1{margin:0;font-size:clamp(26px,4vw,40px)}header p{margin:4px 0 0;color:var(--muted)}
    form,.table{background:color-mix(in srgb,var(--panel) 94%,transparent);border:1px solid var(--line);border-radius:16px;box-shadow:0 18px 60px #0006}
    form{display:grid;grid-template-columns:2fr 1fr 1fr auto;gap:12px;padding:16px;margin-bottom:18px}
    label{display:grid;gap:6px;color:var(--muted);font-size:12px;font-weight:700;text-transform:uppercase;letter-spacing:.08em}
    input,select,button{min-height:44px;border:1px solid var(--line);border-radius:10px;background:#0f1216;color:var(--text);padding:0 12px;font:inherit}
    input:focus,select:focus{outline:2px solid var(--blue);border-color:transparent}button{align-self:end;background:var(--blue);border:0;font-weight:800;cursor:pointer;padding:0 22px}
    .summary{display:flex;justify-content:space-between;gap:12px;color:var(--muted);margin:0 4px 12px}.summary strong{color:var(--text)}
    .table{overflow:auto}table{width:100%;border-collapse:collapse;min-width:720px}th,td{padding:15px 18px;text-align:left;border-bottom:1px solid var(--line)}
    th{color:var(--muted);font-size:11px;text-transform:uppercase;letter-spacing:.1em}tbody tr:hover{background:#ffffff07}tbody tr:last-child td{border-bottom:0}
    .money{color:var(--green);font-weight:800}.name{font-weight:800}.key{display:block;color:var(--muted);font:12px ui-monospace,monospace;margin-top:2px}.empty{text-align:center!important;color:var(--muted);padding:50px!important}
    footer{text-align:center;color:var(--muted);margin-top:18px;font-size:12px}@media(max-width:720px){main{margin-top:22px}form{grid-template-columns:1fr 1fr}.search{grid-column:1/-1}button{grid-column:1/-1}.summary{display:block}}
  </style>
</head>
<body><main>
  <header><div class="logo">⭐</div><div><h1>Clientes frecuentes</h1><p>Benny's Motors · Historial de atenciones y gastos</p></div></header>
  <form method="get" action="/customers">
    <label class="search">Buscar por nombre<input name="name" value="{{.Name}}" placeholder="Nombre, espacios o mayúsculas"></label>
    <label>Últimos días<input type="number" name="days" value="{{if .Days}}{{.Days}}{{end}}" min="0" max="3650" placeholder="Todo"></label>
    <label>Ordenar por<select name="sort">
      <option value="spend" {{if eq .Sort "spend"}}selected{{end}}>Mayor gasto</option>
      <option value="visits" {{if eq .Sort "visits"}}selected{{end}}>Más visitas</option>
      <option value="recent" {{if eq .Sort "recent"}}selected{{end}}>Más reciente</option>
      <option value="name" {{if eq .Sort "name"}}selected{{end}}>Nombre</option>
    </select></label><button type="submit">Aplicar filtros</button>
  </form>
  <div class="summary"><span><strong>{{len .Items}}</strong> clientes encontrados</span><span>{{if .Days}}Periodo: últimos {{.Days}} días{{else}}Periodo: histórico completo{{end}}</span></div>
  <div class="table"><table><thead><tr><th>Cliente</th><th>Gasto</th><th>Visitas</th><th>Atendido por</th><th>Última visita</th></tr></thead><tbody>
    {{range .Items}}<tr><td><span class="name">{{displayName .Name}}</span><span class="key">{{.Name}}</span></td><td class="money">{{money .TotalSpent}}</td><td>{{.Visits}}</td><td>{{.AttendantCount}}</td><td>{{date .LastVisitAt}}</td></tr>
    {{else}}<tr><td colspan="5" class="empty">No hay clientes para estos filtros.</td></tr>{{end}}
  </tbody></table></div><footer>Datos administrados por corps-manager</footer>
</main></body></html>`
