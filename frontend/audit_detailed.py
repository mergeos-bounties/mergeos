#!/usr/bin/env python3
"""Detailed element-level audit for mergeos.shop homepage."""
import json
import os
from playwright.sync_api import sync_playwright

VIEWPORTS = [
    (1440, 900, 'desktop-1440'),
    (1366, 768, 'desktop-1366'),
    (1024, 768, 'tablet-1024'),
    (768, 1024, 'tablet-768'),
    (430, 932, 'mobile-430'),
    (390, 844, 'mobile-390'),
    (360, 800, 'mobile-360'),
]

OUTPUT = 'audit_report'

def detailed_audit(page, width, height, name):
    print(f"\n{'='*60}")
    print(f"DETAILED AUDIT: {name} ({width}x{height})")
    print('='*60)
    
    page.set_viewport_size({'width': width, 'height': height})
    page.goto('https://mergeos.shop/', wait_until='networkidle', timeout=30000)
    page.wait_for_timeout(2000)
    
    issues = []
    
    # Get all major element positions and sizes
    element_data = page.evaluate("""() => {
        const data = {};
        const selectors = [
            '.home-navbar', '.nav-inner', '.brand-link', '.nav-links', '.nav-actions',
            '.public-home-hero', '.public-home-copy', '.public-home-panel',
            '.public-workflow-grid', '.public-talent-strip',
            '.marketplace-actions', '.public-home-copy h1', '.public-home-copy p',
            '.ledger-card-head',
            '.home-shell', '.public-home-page',
            '.primary-button.large', '.secondary-button.large',
            '.public-stat-grid', '.public-talent-list',
            '.brand-link strong'
        ];
        for (const sel of selectors) {
            const el = document.querySelector(sel);
            if (el) {
                const rect = el.getBoundingClientRect();
                const style = window.getComputedStyle(el);
                data[sel] = {
                    tag: el.tagName,
                    width: Math.round(rect.width),
                    height: Math.round(rect.height),
                    top: Math.round(rect.top),
                    left: Math.round(rect.left),
                    right: Math.round(rect.right),
                    visible: rect.width > 0 && rect.height > 0,
                    display: style.display,
                    overflow: style.overflow,
                    textOverflow: style.textOverflow,
                    whiteSpace: style.whiteSpace,
                    text: (el.textContent || '').trim().slice(0, 60),
                    childCount: el.children.length
                };
            } else {
                data[sel] = { error: 'NOT FOUND' };
            }
        }
        return data;
    }""")
    
    # Check for hidden elements
    for sel, info in element_data.items():
        if isinstance(info, dict) and info.get('error'):
            issues.append(f"  MISSING: {sel}")
        elif isinstance(info, dict) and info.get('visible') == False:
            issues.append(f"  HIDDEN: {sel} display={info.get('display')} height={info.get('height')}")
    
    # Check nav-actions on mobile
    if width <= 768:
        # On mobile, the "Log in" button should have proper width
        nav_actions = element_data.get('.nav-actions', {})
        if nav_actions.get('width', 0) < 100:
            issues.append(f"  NARROW NAV-ACTIONS: {nav_actions.get('width')}px")
    
    # Check button text wrapping/overflow
    btn_check = page.evaluate("""() => {
        const issues = [];
        document.querySelectorAll('.primary-button, .secondary-button').forEach(btn => {
            const rect = btn.getBoundingClientRect();
            const text = (btn.textContent || '').trim();
            const style = window.getComputedStyle(btn);
            // Check if button content overflows
            if (btn.scrollWidth > btn.clientWidth + 2) {
                issues.push({
                    type: 'BUTTON_TEXT_OVERFLOW',
                    text: text.slice(0, 40),
                    width: Math.round(rect.width),
                    scrollW: Math.round(btn.scrollWidth),
                    display: style.display,
                    whiteSpace: style.whiteSpace
                });
            }
            // Check very squished buttons
            if (rect.width < 80 && text.length > 5 && rect.width > 0) {
                issues.push({
                    type: 'SQUISHED_BUTTON',
                    text: text.slice(0, 30),
                    width: Math.round(rect.width)
                });
            }
        });
        return issues;
    }""")
    for issue in btn_check:
        issues.append(f"  {issue['type']}: '{issue['text']}' width={issue.get('width')}px")
    
    # Check horizontal scroll
    scroll_w = page.evaluate("document.documentElement.scrollWidth")
    win_w = page.evaluate("window.innerWidth")
    if scroll_w > win_w + 2:
        issues.append(f"  HORIZONTAL SCROLL: docW={scroll_w} > winW={win_w}")
        
        # Find what's overflowing
        overflow_elements = page.evaluate("""() => {
            const issues = [];
            const all = document.querySelectorAll('*');
            for (const el of all) {
                const rect = el.getBoundingClientRect();
                const docW = document.documentElement.scrollWidth;
                if (rect.right > docW + 1 && rect.width > 0 && rect.top < 5000 && rect.bottom > 0) {
                    if (issues.length < 15) {
                        issues.push({
                            tag: el.tagName,
                            cls: (el.className || '').slice(0, 60),
                            id: el.id || '',
                            text: (el.textContent || '').trim().slice(0, 30),
                            right: Math.round(rect.right),
                            docW: docW,
                            width: Math.round(rect.width)
                        });
                    }
                }
            }
            return issues;
        }""")
        for oe in overflow_elements:
            issues.append(f"    <{oe['tag']}> cls='{oe['cls']}' right={oe['right']} > docW={oe['docW']} w={oe['width']} text='{oe['text']}'")
    
    # Check for overlapping elements in critical viewport
    if width <= 768:
        overlap = page.evaluate("""() => {
            const issues = [];
            const critical = document.querySelectorAll('.public-home-hero, .public-home-copy, .public-home-panel, .marketplace-actions, .public-workflow-grid, .public-talent-strip');
            const rects = [];
            critical.forEach(el => {
                const r = el.getBoundingClientRect();
                rects.push({ el: el, tag: el.tagName, cls: (el.className || '').slice(0, 40), rect: r });
            });
            for (let i = 0; i < rects.length; i++) {
                for (let j = i + 1; j < rects.length; j++) {
                    const a = rects[i].rect;
                    const b = rects[j].rect;
                    // Check if one is completely inside another (expected for wrapping)
                    const insideHoriz = a.left >= b.left && a.right <= b.right;
                    const insideVert = a.top >= b.top && a.bottom <= b.bottom;
                    if (!insideHoriz && !insideVert) {
                        const overlapX = Math.max(0, Math.min(a.right, b.right) - Math.max(a.left, b.left));
                        const overlapY = Math.max(0, Math.min(a.bottom, b.bottom) - Math.max(a.top, b.top));
                        if (overlapX > 0 && overlapY > 0) {
                            issues.push(`OVERLAP: ${rects[i].cls} vs ${rects[j].cls} overlap=${overlapX}x${overlapY}`);
                        }
                    }
                }
            }
            return issues.slice(0, 10);
        }""")
        for o in overlap:
            issues.append(f"  {o}")
    
    # Check if hero section elements stack properly on mobile
    if width <= 980:
        hero_style = page.evaluate("""() => {
            const hero = document.querySelector('.public-home-hero');
            if (!hero) return 'NOT_FOUND';
            const style = window.getComputedStyle(hero);
            return style.gridTemplateColumns || style.display;
        }""")
        if '1fr' not in str(hero_style):
            pass  # hero grid template
    
    # Check responsive image/icon sizing
    icon_check = page.evaluate("""() => {
        const issues = [];
        document.querySelectorAll('.brand-mark img').forEach(img => {
            const rect = img.getBoundingClientRect();
            if (rect.width < 20 && rect.width > 0) {
                issues.push(`SMALL BRAND ICON: ${Math.round(rect.width)}x${Math.round(rect.height)}`);
            }
        });
        return issues;
    }""")
    for icon in icon_check:
        issues.append(f"  {icon}")
    
    # Print results
    if issues:
        print(f"  ISSUES FOUND ({len(issues)}):")
        for issue in issues:
            print(f"    {issue}")
    else:
        print("  No issues found!")
    
    # Print element sizes for key sections
    print("\n  Element sizes:")
    for sel, info in element_data.items():
        if isinstance(info, dict) and not info.get('error'):
            print(f"    {sel}: {info.get('width')}x{info.get('height')} visible={info.get('visible')}")
    
    return issues, element_data

def main():
    with sync_playwright() as p:
        browser = p.chromium.launch(headless=True)
        context = browser.new_context()
        page = context.new_page()
        
        all_data = {}
        for width, height, name in VIEWPORTS:
            issues, elements = detailed_audit(page, width, height, name)
            all_data[name] = {'issues': issues, 'elements': elements}
        
        context.close()
        browser.close()
        
        total = sum(len(v['issues']) for v in all_data.values())
        print(f"\n{'='*60}")
        print(f"TOTAL ISSUES: {total}")
        
        # Save full report
        with open(f'{OUTPUT}/full_audit.json', 'w') as f:
            json.dump(all_data, f, indent=2, default=str)

main()
