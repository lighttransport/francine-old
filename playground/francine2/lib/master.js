'use strict';

var Q = require('q');
var jsonrpc = require('multitransport-jsonrpc');
var JsonRpcServer = jsonrpc.server;
var JsonRpcServerTcp = jsonrpc.transports.server.tcp;
var JsonRpcClient = jsonrpc.client;
var JsonRpcClientTcp = jsonrpc.transports.client.tcp;

var LocalInstance = require('./instances/local');
var GceInstance = require('./instances/gce');
var StaticInstanceManager = require('./instance_managers/static');
var QueueScheduler = require('./schedulers/queue');

var initializeRestApi = require('./apis/rest');

var Master = function (argv) {
    var self = this;

    self.instanceType = argv.instanceType || 'local';

    self.restPort = argv.restPort || 3000;
    self.port = argv.port || 5000;
    self.instanceManagerType = argv.instanceManagerType || 'static';
    self.schedulerType = argv.schedulerType || 'queue';
    self.test = argv.test;


    self.spawnInterval = 15 * 1000;
    self.manageInterval = 60 * 1000;
    self.pingInterval = 10 * 1000;

    if (self.test) {
        self.spawnInterval = 0;
        self.manageInterval = 15 * 1000;
    }

    self.instance = null;
    self.instanceManager = null;

    self.scheduler = null;

    self.rpcServer = null;

    self.app = null;

    self.master = null;
    self.workers = null;

    self.finishTaskDefers = {};
    self.seed = 0;
};

Master.prototype.start = function () {
    var self = this;

    // Initialize instance type specific object
    self.initializeInstance();

    // Initialize instance manager
    self.initializeInstanceManager();

    // Initialize scheduler
    self.initializeScheduler();

    // Initialize RPC
    self.initializeRpc();

    // Initialize REST API
    self.initializeRestApi();

    self.loop(self.manageInterval, function () {
        return self.instance.getInstances()
        .then(function (instances) {
            var d = Q.defer();
            self.master = instances.master;
            self.workers = instances.workers;
            self.scheduler.updateWorkers();
            // self.scheduler.schedule();
            d.resolve();
            return d.promise;
        })
        .then(function () {
            return self.instanceManager.manage(self.spawnInterval);
        });
    }).done();

    self.loop(10 * 1000, function () {
        return self.sendPings();
    }).done();

    process.on('uncaughtException', function (error) {
        self.log('Master', error.stack || error);
        process.exit(1);
    });
};

Master.prototype.loop = function (interval, f) {
    return Q().then(function l () { // jshint ignore:line
        return f().delay(interval).then(l);
    });
};

Master.prototype.getPort = function () {
    var self = this;
    return self.port;
};

Master.prototype.getRestPort = function () {
    var self = this;
    return self.restPort;
};

Master.prototype.getMaster = function () {
    var self = this;
    return self.master;
};

Master.prototype.getWorkers = function () {
    var self = this;
    return self.workers;
};

Master.prototype.log = function (from, message) {
    console.log('Francine: ' + from + ': ' + message);
};

Master.prototype.initializeInstance = function () {
    var self = this;

    switch (self.instanceType) {
        case 'local':
            self.instance = new LocalInstance(self);
            break;
        case 'gce':
            self.instance = new GceInstance(self);
            break;
    }

    if (!self.instance) {
        self.log('Master', 'Error: Invalid worker instance type ' + self.instanceType);
        process.exit(1);
    }

    self.log('Master', 'Worker instance type: ' + self.instanceType);
};

Master.prototype.initializeInstanceManager = function () {
    var self = this;

    switch (self.instanceManagerType) {
        case 'static':
            self.instanceManager = new StaticInstanceManager(self, self.instance, 4);
            break;
    }

    if (!self.instanceManager) {
        self.log('Master', 'Error: Invalid instance manager type ' + self.instanceManagerType);
        process.exit(1);
    }

    self.log('Master', 'Instance manager type: ' + self.instanceManagerType);
};

Master.prototype.initializeRpc = function () {
    var self = this;

    self.log('Master', 'Waiting on port ' + self.port + ' for JSON RPC request...');

    self.rpcServer = new JsonRpcServer(new JsonRpcServerTcp(self.port), {
        pong: function (info, callback) {
            // self.log('Master', 'Pong received from ' + info.workerName);
            self.dispatchPong(info);
            callback(null, {});
        },
        finish: function (info, callback) {
            self.log('Master', 'Finish received from ' + info.workerName);
            self.dispatchFinish(info);
            callback(null, {});
        },
    });
};

Master.prototype.initializeScheduler = function () {
    var self = this;

    switch (self.schedulerType) {
        case 'queue':
            self.scheduler = new QueueScheduler(self, self.instance);
            break;
    }

    if (!self.scheduler) {
        self.log('Master', 'Error: invalid scheduler type ' + self.schedulerType);
        process.exit(1);
    }

    self.log('Master', 'Scheduler type: ' + self.schedulerType);
};

Master.prototype.initializeRestApi = function () {
    var self = this;
    self.app = initializeRestApi(self, self.scheduler);
};

Master.prototype.sendPings = function () {
    var self = this;
    var d = Q.defer();

    self.log('Master', 'Sending pings...');

    for (var key in self.workers) {
        if (self.workers.hasOwnProperty(key)) {
            self.sendPing(self.workers[key]);
        }
    }

    d.resolve();
    return d.promise;
};

Master.prototype.sendPing = function (worker) {
    var self = this;

    self.log('Master', 'Send ping to ' + worker.name + '...');

    var client = new JsonRpcClient(new JsonRpcClientTcp(worker.host, worker.port));
    client.register('ping');
    client.ping({
        workerName: worker.name,
        master: self.getMaster()
    }, function () {
        client.shutdown();
    });
};


Master.prototype.dispatchPong = function (info) {
    var self = this;

    info.logs.map(function (message) {
        self.log('[' + info.workerName + '] ' + message.from, message.message);
    });
};

Master.prototype.delayUntilFinishTask = function (taskName) {
    var self = this;
    var d = Q.defer();
    self.finishTaskDefers[taskName] = d;
    return d.promise;
    // TODO(peryaudo): write timeout
};

Master.prototype.dispatchFinish = function (info) {
    var self = this;

    if (info.type === 'TASK') {
        var d = self.finishTaskDefers[info.task.name];
        delete self.finishTaskDefers[info.task.name];
        if (d) {
            d.resolve(info);
        }
        // TODO(peryaudo): write error handling
    }

    self.scheduler.dispatchFinish(info);
};

Master.prototype.getId = function () {
    var self = this;
    self.seed++;
    return (Date.now() | 0) + '-' + self.seed;
};

module.exports = Master;
