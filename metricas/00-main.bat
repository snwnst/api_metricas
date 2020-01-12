net start | find "API_METRICAS" 
if ERRORLEVEL 1 net stop "API_METRICAS" 
net start "API_METRICAS" 