if ( $global.PouchDB === undefined ) {
    try {
        $global.PouchDB = require('pouchdb');
    } catch(e) {
        throw("kivik: pouchdb bindings: Cannot find global PouchDB object. Did you load the PouchDB library?");
    }
}
try {
    require('pouchdb-all-dbs')($global.PouchDB);
} catch(e) {}

try {
    $global.PouchDB.plugin(require('pouchdb-find'));
} catch(e) {}

try {
    global.XMLHttpRequest = require('xhr2');
} catch(e) {}
