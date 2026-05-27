#!/usr/bin/env python3
"""Audit MergeOS homepage responsive layout at multiple viewport widths"""
from playwright.sync_api import sync_playwright

URL = "http://127.0.0.1:5173"
VIEWPORTS = [
    (1440, 900, "desktop_1440"),
    (1366, 768, "desktop_1366"),
    (1024, 768, "tablet_1024"),
    (768, 1024, "tablet_768"),
    (430, 932, "mobile_430"),
    (390, 844, "mobile_390"),
    (360, 780, "mobile_360"),
]

import os
os.makedirs("/c/Users/25936/.hermes/image_cache/responsive_audit", exist_ok=True)

with sync_playwright() as p:
    b = p.chromium.launch(headless=True, args=["--no-sandbox"], timeout=10000)
    
    for w, h, name in VIEWPORTS:
        page = b.new_page(viewport={"width": w, "height": h})
        try:
            page.goto(URL, timeout=15000, wait_until="networkidle")
            page.wait_for_timeout(2000)
            path = f"/c/Users/25936/.hermes/image_cache/responsive_audit/{name}.png"
            page.screenshot(path=path, full_page=True)
            print(f"[OK] {name} ({w}x{h}) - saved")
            
            # Check for horizontal overflow
            overflow = page.evaluate("""() => {
                const maxW = document.body.scrollWidth;
                const vpW = window.innerWidth;
                return {scroll: maxW, viewport: vpW, overflows: maxW > vpW + 5};
            }""")
            if overflow["overflows"]:
                print(f"  ⚠️ HORIZONTAL OVERFLOW: scroll={overflow['scroll']} vp={overflow['viewport']}")
            else:
                print(f"  ✅ No overflow: scroll={overflow['scroll']} vp={overflow['viewport']}")
                
        except Exception as e:
            print(f"[FAIL] {name}: {str(e)[:60]}")
        finally:
            page.close()
    
    b.close()
    print("\nDONE")
