const fs = require('fs');

module.exports = {
  input: ['src/**/*.{js,jsx,ts,tsx}', '!**/node_modules/**', '!src/**/*.test.{ts,tsx}'],
  output: './',
  options: {
    debug: true,
    func: {
      list: ['t', 'i18n.t'],
      extensions: ['.js', '.jsx', '.ts', '.tsx']
    },
    lngs: ['zh'],
    ns: ['translation'],
    defaultLng: 'zh',
    defaultNs: 'translation',
    resource: {
      loadPath: 'src/i18n/resources/{{lng}}/{{ns}}.json',
      savePath: 'src/i18n/resources/{{lng}}/{{ns}}.json',
      jsonIndent: 2,
      lineEnding: '\n'
    },
    nsSeparator: false, // namespace separator
    keySeparator: false, // key separator
    interpolation: {
      prefix: '{{',
      suffix: '}}'
    }
  },
  transform: function customTransform(file, enc, done) {
    const parser = this.parser;
    const content = fs.readFileSync(file.path, enc);

    parser.parseFuncFromString(content, (key, options) => {
      options.defaultValue = key;
      parser.set(key, options);
    });

    done();
  }
};
