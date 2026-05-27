#!/usr/bin/env python3
"""Take full-page screenshots of mergeos.shop at 7 viewport widths."""
import asyncio
import os
from playwright.async_api import async_playwright

VIEWPORTS = [
    (1440, 900, 'desktop-1440'),
    (1366, 768, 'desktop-1366'),
    (1024, 768, 'tablet-landscape-1024'),
    (768, 1024, 'tablet-portrait-768'),
    (430, 932, 'mobile-430'),
    (390, 844, 'mobile-390'),
    (360, 800, 'mobile-360'),
]

OUTPUT_DIR = 'screenshots'
os.makedirs(OUTPUT_DIR, exist_ok=True)

async def main():
    async with async_playwright() as p:
        browser = await p.chromium.launch(headless=True)
        
        for width, height, name in VIEWPORTS:
            context = await browser.new_context(
                viewport={'width': width, 'height': height},
                device_scale_factor=2,
            )
            page = await context.new_page()
            
            try:
                print(f"Navigating to {name} ({width}x{height})...")
                await page.goto('https://mergeos.shop/', wait_until='networkidle', timeout=30000)
                await page.wait_for_timeout(2000)
                
                # Take full-page screenshot
                filename = f'{OUTPUT_DIR}/before-{name}.png'
                await page.screenshot(path=filename, full_page=True)
                print(f"  Saved {filename}")
                
                # Also check viewport screenshot for visible issues
                vp_filename = f'{OUTPUT_DIR}/before-{name}-vp.png'
                await page.screenshot(path=vp_filename, full_page=False)
                print(f"  Saved {vp_filename}")
                
            except Exception as e:
                print(f"  Error: {e}")
            
            await context.close()
        
        await browser.close()

asyncio.run(main())
