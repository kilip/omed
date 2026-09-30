Omed PWA / favicon icon set
============================
theme color: #2563EB

Files:
- favicon.ico            -> place at web root, link in <head> as favicon
- icon-16.png / icon-32.png / icon-48.png -> favicon-16x16.png / favicon-32x32.png
- apple-touch-icon.png (180x180) -> <link rel="apple-touch-icon" href="/apple-touch-icon.png">
- icon-192.png, icon-512.png -> standard PWA manifest icons (purpose: any)
- icon-maskable-192.png, icon-maskable-512.png -> PWA manifest icons (purpose: maskable)
- omed-mark.svg -> source vector (transparent bg)
- manifest-icons-snippet.json -> drop into your PWA manifest.json's "icons" array

Suggested <head> tags:
<link rel="icon" href="/favicon.ico" sizes="any">
<link rel="icon" href="/icon.svg" type="image/svg+xml">
<link rel="apple-touch-icon" href="/apple-touch-icon.png">
