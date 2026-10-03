import { functions } from './parse-api';

const extras = [
  '_malloc',
  '_free',
  '_Z3_set_error_handler',
];

const legacyInterpolationExports = [
  '_Z3_mk_interpolant',
  '_Z3_mk_interpolation_context',
  '_Z3_get_interpolant',
  '_Z3_compute_interpolant',
  '_Z3_interpolation_profile',
  '_Z3_read_interpolation_problem',
  '_Z3_check_interpolant',
  '_Z3_write_interpolation_problem',
];

const fns = functions.map(f => '_' + f.name);
console.log(JSON.stringify([...extras, ...fns, ...legacyInterpolationExports]));
