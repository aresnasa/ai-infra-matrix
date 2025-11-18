#!/usr/bin/env python3
"""
ai-infra-matrix Favicon Generator
生成各种尺寸和格式的favicon图标，支持动态子页面图标
"""

import os
from PIL import Image, ImageDraw, ImageFont
import json

def create_ai_matrix_favicon():
    """创建ai-infra-matrix主图标"""
    # 创建256x256的基础图标
    size = 256
    img = Image.new('RGBA', (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)
    
    # 绘制背景渐变（从深蓝到浅蓝）
    for y in range(size):
        # 渐变色从深蓝#1a1a2e到亮蓝#16213e到科技蓝#0f3460
        ratio = y / size
        r = int(26 * (1-ratio) + 15 * ratio)
        g = int(26 * (1-ratio) + 52 * ratio) 
        b = int(46 * (1-ratio) + 96 * ratio)
        color = (r, g, b, 255)
        draw.line([(0, y), (size, y)], fill=color)
    
    # 绘制AI矩阵风格的图案
    # 中心圆圈代表AI核心
    center = size // 2
    radius = size // 6
    draw.ellipse([center-radius, center-radius, center+radius, center+radius], 
                fill=(100, 200, 255, 200))
    
    # 绘制连接线网络（代表基础设施）
    grid_size = 8
    spacing = size // grid_size
    line_color = (80, 180, 255, 150)
    
    # 垂直线
    for i in range(1, grid_size):
        x = i * spacing
        draw.line([(x, spacing), (x, size-spacing)], fill=line_color, width=2)
    
    # 水平线
    for i in range(1, grid_size):
        y = i * spacing
        draw.line([(spacing, y), (size-spacing, y)], fill=line_color, width=2)
    
    # 绘制节点（代表服务节点）
    node_color = (150, 220, 255, 200)
    node_radius = 6
    nodes = [
        (spacing*2, spacing*2), (spacing*6, spacing*2),
        (spacing*2, spacing*6), (spacing*6, spacing*6),
        (spacing*4, spacing*4)  # 中心节点
    ]
    
    for x, y in nodes:
        draw.ellipse([x-node_radius, y-node_radius, x+node_radius, y+node_radius], 
                    fill=node_color)
    
    # 绘制AI字样（简化版）
    try:
        # 尝试使用系统字体
        font_size = size // 8
        try:
            font = ImageFont.truetype("/System/Library/Fonts/Arial.ttf", font_size)
        except:
            font = ImageFont.load_default()
        
        text = "AI"
        bbox = draw.textbbox((0, 0), text, font=font)
        text_width = bbox[2] - bbox[0]
        text_height = bbox[3] - bbox[1]
        
        x = (size - text_width) // 2
        y = size // 2 + radius + 20
        
        # 添加文字阴影
        draw.text((x+2, y+2), text, font=font, fill=(0, 0, 0, 100))
        # 主文字
        draw.text((x, y), text, font=font, fill=(255, 255, 255, 255))
        
    except Exception as e:
        print(f"字体渲染错误: {e}")
    
    return img

def create_sub_page_icons():
    """创建子页面专用图标"""
    icons = {}
    
    # JupyterHub图标 - 橙色主题
    jupyter_icon = Image.new('RGBA', (64, 64), (0, 0, 0, 0))
    draw = ImageDraw.Draw(jupyter_icon)
    
    # 橙色渐变背景
    for y in range(64):
        ratio = y / 64
        r = int(255 * (1-ratio) + 230 * ratio)
        g = int(150 * (1-ratio) + 100 * ratio)
        b = int(50 * (1-ratio) + 20 * ratio)
        color = (r, g, b, 255)
        draw.line([(0, y), (64, y)], fill=color)
    
    # 绘制Jupyter标志性的三个圆圈
    circles = [(16, 32), (32, 16), (48, 32)]
    for x, y in circles:
        draw.ellipse([x-6, y-6, x+6, y+6], fill=(255, 255, 255, 200))
    
    icons['jupyter'] = jupyter_icon
    
    # Kubernetes图标 - 蓝色主题
    k8s_icon = Image.new('RGBA', (64, 64), (0, 0, 0, 0))
    draw = ImageDraw.Draw(k8s_icon)
    
    # 蓝色渐变背景
    for y in range(64):
        ratio = y / 64
        r = int(50 * (1-ratio) + 100 * ratio)
        g = int(150 * (1-ratio) + 200 * ratio)
        b = int(255 * (1-ratio) + 230 * ratio)
        color = (r, g, b, 255)
        draw.line([(0, y), (64, y)], fill=color)
    
    # 绘制K8s标志性的轮子形状
    center = 32
    radius = 20
    for i in range(8):
        angle = i * 45
        x1 = center + radius * 0.5
        y1 = center
        x2 = center + radius
        y2 = center
        draw.line([(x1, y1), (x2, y2)], fill=(255, 255, 255, 200), width=3)
    
    draw.ellipse([center-8, center-8, center+8, center+8], fill=(255, 255, 255, 200))
    
    icons['kubernetes'] = k8s_icon
    
    # Ansible图标 - 红色主题
    ansible_icon = Image.new('RGBA', (64, 64), (0, 0, 0, 0))
    draw = ImageDraw.Draw(ansible_icon)
    
    # 红色渐变背景
    for y in range(64):
        ratio = y / 64
        r = int(220 * (1-ratio) + 180 * ratio)
        g = int(50 * (1-ratio) + 30 * ratio)
        b = int(50 * (1-ratio) + 30 * ratio)
        color = (r, g, b, 255)
        draw.line([(0, y), (64, y)], fill=color)
    
    # 绘制Ansible标志性的A字形
    points = [(32, 10), (20, 50), (25, 50), (32, 25), (39, 50), (44, 50)]
    draw.polygon(points, fill=(255, 255, 255, 200))
    
    icons['ansible'] = ansible_icon
    
    # 管理员图标 - 绿色主题
    admin_icon = Image.new('RGBA', (64, 64), (0, 0, 0, 0))
    draw = ImageDraw.Draw(admin_icon)
    
    # 绿色渐变背景
    for y in range(64):
        ratio = y / 64
        r = int(50 * (1-ratio) + 100 * ratio)
        g = int(200 * (1-ratio) + 150 * ratio)
        b = int(100 * (1-ratio) + 50 * ratio)
        color = (r, g, b, 255)
        draw.line([(0, y), (64, y)], fill=color)
    
    # 绘制管理员图标（齿轮）
    center = 32
    outer_radius = 18
    inner_radius = 12
    teeth = 8
    
    for i in range(teeth):
        angle1 = i * 45
        angle2 = (i + 0.5) * 45
        
        # 外齿
        x1 = center + outer_radius
        y1 = center
        x2 = center + (outer_radius + 4)
        y2 = center
        draw.line([(x1, y1), (x2, y2)], fill=(255, 255, 255, 200), width=4)
    
    draw.ellipse([center-inner_radius, center-inner_radius, 
                 center+inner_radius, center+inner_radius], 
                 fill=(255, 255, 255, 200))
    draw.ellipse([center-6, center-6, center+6, center+6], fill=(0, 0, 0, 0))
    
    icons['admin'] = admin_icon
    
    return icons

def generate_favicon_sizes(base_image):
    """生成各种尺寸的favicon"""
    sizes = [16, 32, 48, 64, 128, 256]
    favicons = {}
    
    for size in sizes:
        favicon = base_image.resize((size, size), Image.Resampling.LANCZOS)
        favicons[size] = favicon
    
    return favicons

def save_favicon_files():
    """保存favicon文件"""
    # 获取当前脚本目录
    current_dir = os.path.dirname(os.path.abspath(__file__))
    
    print("🚀 开始生成ai-infra-matrix图标...")
    
    # 创建主图标
    main_icon = create_ai_matrix_favicon()
    
    # 生成各种尺寸
    favicons = generate_favicon_sizes(main_icon)
    
    # 保存ICO文件（包含多种尺寸）
    ico_sizes = [16, 32, 48]
    ico_images = [favicons[size] for size in ico_sizes]
    ico_images[0].save(
        os.path.join(current_dir, 'favicon.ico'),
        format='ICO',
        sizes=[(size, size) for size in ico_sizes]
    )
    print("✅ favicon.ico 已生成")
    
    # 保存PNG文件
    for size in [16, 32, 192, 512]:
        if size in favicons:
            favicons[size].save(
                os.path.join(current_dir, f'favicon-{size}x{size}.png'),
                format='PNG'
            )
            print(f"✅ favicon-{size}x{size}.png 已生成")
    
    # 保存高质量SVG版本（手动创建）
    svg_content = '''<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256">
  <defs>
    <linearGradient id="bg" x1="0%" y1="0%" x2="0%" y2="100%">
      <stop offset="0%" style="stop-color:#1a1a2e;stop-opacity:1" />
      <stop offset="50%" style="stop-color:#16213e;stop-opacity:1" />
      <stop offset="100%" style="stop-color:#0f3460;stop-opacity:1" />
    </linearGradient>
    <linearGradient id="center" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#64c8ff;stop-opacity:0.8" />
      <stop offset="100%" style="stop-color:#50a3ff;stop-opacity:0.6" />
    </linearGradient>
  </defs>
  
  <!-- 背景 -->
  <rect width="256" height="256" fill="url(#bg)" rx="32"/>
  
  <!-- 网格线 -->
  <g stroke="#50b4ff" stroke-width="2" stroke-opacity="0.6" fill="none">
    <!-- 垂直线 -->
    <line x1="64" y1="32" x2="64" y2="224"/>
    <line x1="128" y1="32" x2="128" y2="224"/>
    <line x1="192" y1="32" x2="192" y2="224"/>
    <!-- 水平线 -->
    <line x1="32" y1="64" x2="224" y2="64"/>
    <line x1="32" y1="128" x2="224" y2="128"/>
    <line x1="32" y1="192" x2="224" y2="192"/>
  </g>
  
  <!-- 中心AI核心 -->
  <circle cx="128" cy="128" r="42" fill="url(#center)"/>
  
  <!-- 节点 -->
  <g fill="#96dcff" fill-opacity="0.8">
    <circle cx="64" cy="64" r="6"/>
    <circle cx="192" cy="64" r="6"/>
    <circle cx="64" cy="192" r="6"/>
    <circle cx="192" cy="192" r="6"/>
    <circle cx="128" cy="128" r="8"/>
  </g>
  
  <!-- AI文字 -->
  <text x="128" y="200" font-family="Arial, sans-serif" font-size="32" font-weight="bold" 
        text-anchor="middle" fill="white" fill-opacity="0.9">AI</text>
</svg>'''
    
    with open(os.path.join(current_dir, 'favicon.svg'), 'w', encoding='utf-8') as f:
        f.write(svg_content)
    print("✅ favicon.svg 已生成")
    
    # 创建子页面图标
    print("\n🎨 创建子页面图标...")
    sub_icons = create_sub_page_icons()
    
    for name, icon in sub_icons.items():
        # 保存PNG格式
        icon.save(os.path.join(current_dir, f'icon-{name}.png'), format='PNG')
        print(f"✅ icon-{name}.png 已生成")
    
    # 创建图标映射配置文件
    icon_config = {
        "default": "favicon.ico",
        "pages": {
            "jupyter": "icon-jupyter.png",
            "kubernetes": "icon-kubernetes.png", 
            "ansible": "icon-ansible.png",
            "admin": "icon-admin.png"
        },
        "routes": {
            "/projects": "icon-jupyter.png",
            "/admin": "icon-admin.png",
            "/kubernetes": "icon-kubernetes.png",
            "/ansible": "icon-ansible.png"
        }
    }
    
    with open(os.path.join(current_dir, 'favicon-config.json'), 'w', encoding='utf-8') as f:
        json.dump(icon_config, f, indent=2, ensure_ascii=False)
    print("✅ favicon-config.json 配置文件已生成")
    
    print("\n🎉 所有图标文件生成完成！")
    print("\n📁 生成的文件:")
    generated_files = [
        "favicon.ico", "favicon.svg", "favicon-config.json",
        "favicon-16x16.png", "favicon-32x32.png", 
        "favicon-192x192.png", "favicon-512x512.png",
        "icon-jupyter.png", "icon-kubernetes.png", 
        "icon-ansible.png", "icon-admin.png"
    ]
    
    for file in generated_files:
        print(f"  • {file}")

if __name__ == "__main__":
    save_favicon_files()
