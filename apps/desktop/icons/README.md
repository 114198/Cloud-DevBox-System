# Application Icons

This directory should contain the application icons in various formats and sizes.

## Required Icons

### Windows
- `icon.ico` - Windows application icon (multi-resolution ICO file)

### macOS
- `icon.icns` - macOS application icon (ICNS format)

### Linux & General
- `32x32.png` - 32x32 pixel PNG icon
- `128x128.png` - 128x128 pixel PNG icon
- `128x128@2x.png` - 256x256 pixel PNG icon (for Retina displays)
- `icon.png` - Default icon (recommended 512x512 or 1024x1024)

## Icon Generation

You can use tools like:
- [tauri-icon](https://github.com/nicholasio/tauri-icon) - Generate all required icons from a single source
- [IconKitchen](https://icon.kitchen/) - Online icon generator
- [Figma](https://figma.com) - Design tool with export capabilities

### Using tauri-icon

```bash
npm install -g tauri-icon
tauri-icon --input source-icon.png --output ./icons
```

## Design Guidelines

- Use a simple, recognizable design
- Ensure the icon is visible at small sizes
- Use the Cloud DevBox brand colors
- Include transparency where appropriate
