import sharp from 'sharp'
import fs from 'fs'
import path from 'path'

const svgIcon = `
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">
  <rect width="512" height="512" fill="#667eea"/>
  <text x="256" y="380" font-size="200" font-weight="bold" fill="white" text-anchor="middle" dominant-baseline="middle">C</text>
</svg>
`

const publicDir = path.resolve(process.cwd(), 'public')

async function generateIcons() {
  try {
    // Generate 192x192 icon
    await sharp(Buffer.from(svgIcon))
      .resize(192, 192, { fit: 'contain', background: { r: 102, g: 126, b: 234, alpha: 1 } })
      .png()
      .toFile(path.join(publicDir, 'pwa-192x192.png'))

    console.log('✓ Created pwa-192x192.png')

    // Generate 512x512 icon
    await sharp(Buffer.from(svgIcon))
      .resize(512, 512, { fit: 'contain', background: { r: 102, g: 126, b: 234, alpha: 1 } })
      .png()
      .toFile(path.join(publicDir, 'pwa-512x512.png'))

    console.log('✓ Created pwa-512x512.png')
  } catch (error) {
    console.error('Error generating icons:', error)
    process.exit(1)
  }
}

generateIcons()
