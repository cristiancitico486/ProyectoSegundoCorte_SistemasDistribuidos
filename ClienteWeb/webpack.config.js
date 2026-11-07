// webpack.config.js
const path = require('path');

module.exports = {
  entry: './cliente_receptor_fragmentos.js',
  mode: 'development',
  devtool: false,
  output: {
    filename: 'bundle.js',
    path: path.resolve(__dirname, './'),
  },
};
