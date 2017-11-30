function _interopDefault (ex) { return (ex && (typeof ex === 'object') && 'default' in ex) ? ex['default'] : ex; }
var inherits = _interopDefault(require('inherits'));
inherits(PouchError, Error);

function PouchError(status, error, reason) {
    Error.call(this, reason);
    this.status = status;
    this.name = error;
    this.message = reason;
    this.error = true;
}

PouchError.prototype.toString = function () {
    return JSON.stringify({
        status: this.status,
        name: this.name,
        message: this.message,
        reason: this.reason
    });
};

$global.ReconstitutePouchError = function(str) {
    const o = JSON.parse(str);
    Object.setPrototypeOf(o, PouchError.prototype);
    return o;
};
