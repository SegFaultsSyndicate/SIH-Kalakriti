import { describe, it, expect } from 'vitest';
import * as fs from 'node:fs';
import * as path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

describe('OpenAPI to operations.ts parity', () => {
  const openApiPath = path.resolve(__dirname, '../../../../services/bff/openapi.json');
  const operationsFilePath = path.resolve(__dirname, 'operations.ts');

  it('verifies that every OpenAPI path has a corresponding typed operation in operations.ts', () => {
    const openApiRaw = fs.readFileSync(openApiPath, 'utf8');
    const openApi = JSON.parse(openApiRaw);
    const operationsCode = fs.readFileSync(operationsFilePath, 'utf8');

    const paths = Object.keys(openApi.paths || {});
    expect(paths.length).toBeGreaterThan(0);

    const missingPaths: string[] = [];

    for (const apiPath of paths) {
      // 1. Direct type reference: paths['/my/path']
      const pathTypeReference = `paths['${apiPath}']`;
      if (operationsCode.includes(pathTypeReference)) {
        continue;
      }

      // 2. Direct string literal: '/my/path'
      if (operationsCode.includes(`'${apiPath}'`)) {
        continue;
      }

      // 3. Dynamic route pattern in template literal, e.g. `/listings/${encodeURIComponent(id)}/submit`
      // Check that every static path segment is present in operations.ts
      const segments = apiPath.split('/').filter((s) => s && !s.startsWith('{'));
      const allSegmentsPresent = segments.length > 0 && segments.every((seg) => operationsCode.includes(seg));

      if (!allSegmentsPresent) {
        missingPaths.push(apiPath);
      }
    }

    expect(missingPaths).toEqual([]);
  });
});
