const path = require('path');
const fs = require('fs');

// Require EJS from the titles-server node_modules to avoid installing it locally.
const ejsPath = '/Users/mokshchadha/work/pippin/titles-server/node_modules/ejs';
let ejs;
try {
  ejs = require(ejsPath);
} catch (err) {
  // Fallback: try to require from local node_modules
  try {
    ejs = require('ejs');
  } catch (e) {
    console.error('Error: EJS not found. Please install it or ensure titles-server node_modules exists.');
    process.exit(1);
  }
}

// Read payload from stdin
const rawData = fs.readFileSync(0, 'utf-8');
let input;
try {
  input = JSON.parse(rawData);
} catch (err) {
  console.error('Error parsing stdin JSON:', err);
  process.exit(1);
}

const { templatePath, orders, reportData } = input;

ejs.renderFile(templatePath, { orders, reportData }, (err, html) => {
  if (err) {
    console.error('EJS Render Error:', err);
    process.exit(1);
  }
  process.stdout.write(html);
});
