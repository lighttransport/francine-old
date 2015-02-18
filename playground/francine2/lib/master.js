'use strict';

var Q = require('q');
var fs = require('fs');
var request = require('request');
var jsonrpc = require('multitransport-jsonrpc');

var JsonRpcServer = jsonrpc.server;
var JsonRpcServerTcp = jsonrpc.transports.server.tcp;
var JsonRpcClient = jsonrpc.client;
var JsonRpcClientTcp = jsonrpc.transports.client.tcp;

var LocalInstance = require('./instances/local');
var GceInstance = require('./instances/gce');
var StaticInstanceManager = require('./instance_managers/static');
var QueueScheduler = require('./schedulers/queue');

var DropboxResource = require('./resources/dropbox');

var initializeRestApi = require('./apis/rest');

var Master = function (argv) {
    var self = this;

    self.instanceType = argv.instanceType || 'local';

    self.restPort = argv.restPort || 3000;
    self.port = argv.port || 5000;
    self.instanceManagerType = argv.instanceManagerType || 'static';
    self.schedulerType = argv.schedulerType || 'queue';
    self.test = argv.test;

    self.spawnInterval = 500;
    self.manageInterval = 15 * 1000;
    self.pingInterval = 10 * 1000;

    if (self.test) {
        self.spawnInterval = 0;
        self.manageInterval = 5 * 1000;
    }

    self.instance = null;
    self.instanceManager = null;

    self.scheduler = null;

    self.rpcServer = null;

    self.app = null;

    self.master = null;
    self.workers = null;

    self.finishTaskDefers = {};
    self.finishFetchingDefers = {};
    self.seed = 0;

    self.configs = {};
    self.resources = null;

    self.sessions = {};
};

Master.prototype.start = function () {
    var self = this;

    var francinerc = process.env.HOME + '/.francinerc';
    if (fs.existsSync(francinerc)) {
        self.configs = JSON.parse(fs.readFileSync(francinerc, { encoding: 'utf-8' }));
        self.log('Master', 'Read configuration form .fracinerc');
    } else {
        self.log('Master', 'No .fracinerc available.');
    }

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

    // Initialize resources
    self.initializeResources();

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

//
// Getters
//

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

Master.prototype.getId = function () {
    var self = this;
    self.seed++;
    return (Date.now() | 0) + '-' + self.seed;
};

Master.prototype.getResourceToken = function (resourceType) {
    var self = this;
    return self.resources[resourceType].getToken();
};

Master.prototype.getSession = function () {
    var self = this;
    return self.sessions;
};

Master.prototype.getNextCachedWorker = function (sessionName) {
    var self = this;

    // Take a worker with the session resources from top and shift it back.
    var workerName = self.sessions[sessionName].cachedWorkers.pop();
    self.sessions[sessionName].cachedWorkers.unshift(workerName);

    return self.workers[workerName];
};

//
// Logger
//

Master.prototype.log = function (from, message) {
    console.log('Francine: ' + from + ': ' + message);
};

//
// Initializers
//

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
            self.instanceManager = new StaticInstanceManager(self, self.instance, 8);
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
            // self.log('Master', 'Finish received from ' + info.workerName);
            self._dispatchFinish(info);
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
    self.app = initializeRestApi(self);
};

Master.prototype.initializeResources = function () {
    var self = this;
    self.resources = {};
    self.resources.dropbox = new DropboxResource();
    self.resources.dropbox.initializeInMaster(self, self.configs.dropbox);
    self.resources.dropbox.registerEndpoint(self.app);
};

Master.prototype.sendPings = function () {
    var self = this;
    var d = Q.defer();

    if (self.workers) {
        self.log('Master', 'Sending pings to ' + Object.keys(self.workers).length + ' workers...');

        for (var key in self.workers) {
            if (self.workers.hasOwnProperty(key)) {
                self.sendPing(self.workers[key]);
            }
        }
    }

    d.resolve();
    return d.promise;
};

//
// Ping / Pong management
//

Master.prototype.sendPing = function (worker) {
    var self = this;

    // self.log('Master', 'Send ping to ' + worker.name + '...');

    var client = new JsonRpcClient(new JsonRpcClientTcp(worker.host, worker.port, { timeout: 10, retries: 0 }));
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

//
// Tasks
//

Master.prototype.runTask = function (workerName, task) {
    var self = this;

    var worker = self.workers[workerName];

    var client = new JsonRpcClient(new JsonRpcClientTcp(worker.host, worker.port, { timeout: 10, retries: 0 }));
    client.register('run');
    //self.master.log('Master', 'Task ' + task.name + ' of ' + task.type + ' sent');
    client.run(task, function () {
        client.shutdown();
    });
};

Master.prototype.delayUntilFinishTask = function (taskName) {
    var self = this;
    var d = Q.defer();
    self.finishTaskDefers[taskName] = d;
    return d.promise;
    // TODO(peryaudo): write timeout
};

Master.prototype.delayUntilFinishFetching = function (taskName) {
    var self = this;
    var d = Q.defer();
    self.finishFetchingDefers[taskName] = d;
    return d.promise;
    // TODO(peryaudo): write timeout
};

//
// Finish management
//

Master.prototype._dispatchFinish = function (info) {
    var self = this;
    var d;

    self.scheduler.dispatchFinish(info);

    if (info.type === 'TASK') {
        d = self.finishTaskDefers[info.task.name];
        delete self.finishTaskDefers[info.task.name];
        // self.log('Master', Object.keys(self.finishTaskDefers).length + ' defers waiting for dispatch after ' + info.task.name + ' of ' + info.task.type);
        if (d) {
            d.resolve(info);
        }
        // TODO(peryaudo): write error handling
    } else if (info.type === 'FETCHING') {
        info.cachedSessions.map(function (cachedSession) {
            // TODO(peryaudo): delete session garbage collection
            if (!self.sessions[cachedSession]) {
                return;
            }

            // TODO(peryaudo): it is inefficient
            if (self.sessions[cachedSession].cachedWorkers.indexOf(info.workerName) < 0) {
                self.sessions[cachedSession].cachedWorkers.push(info.workerName);
            }
        });

        d = self.finishFetchingDefers[info.taskName];
        delete self.finishFetchingDefers[info.taskName];
        if (d) {
            d.resolve(info);
        }
    }
};

//
// Session / Execution management
//

Master.prototype.createSession = function (options) {
    var self = this;

    var sessionName = 'session' + self.getId();

    self.sessions[sessionName] = {
        name: sessionName,
        options: {
            resources: options.resources
        },
        cachedWorkers: []
    };

    return sessionName;
};

Master.prototype.deleteSession = function (sessionName) {
    var self = this;

    delete self.sessions[sessionName];
};

Master.prototype.createExecution = function (options) {
    var self = this;
    var d = Q.defer();

    if (!self.sessions.hasOwnProperty(options.sessionName)) {
        self.log('Master', 'No such session available! ' + options.sessionName);
        d.reject();
        return d.promise;
    }

    var session = self.sessions[options.sessionName];

    var executionName = 'execution' + self.getId();

    self.log('Master', 'Execution ' + executionName + ' created.');

    var execution = {
        name: executionName,
        options: options,
        startTime: Date.now() | 0,
        tasks: []
    };

    var initialProducingTaskName = self.scheduler.createProducingTask(session, execution, 0);
    self.scheduler.schedule();

    return self.delayUntilFinishFetching(initialProducingTaskName)
    .then(function () {
        self.log('Master', 'finished fetching.');
        var producingTaskNames = [initialProducingTaskName];
        for (var i = 1; i < execution.options.parallel; i++) {
            producingTaskNames.push(self.scheduler.createProducingTask(session, execution, i));
        }
        self.scheduler.schedule();

        var producingTasks = producingTaskNames.map(function (producingTaskName) {
            return self.delayUntilFinishTask(producingTaskName);
        });

        // Do two layer reducing iff. the number of producing tasks is more than 4
        if (producingTasks.length <= 4) {
            return Q.all(producingTasks);
        } else {
            return self._createIntermediateReducing(session, execution, producingTasks);
        }
    })
    .then(function (producings) {
        var reducingTaskName = self.scheduler.createReducingTask(session, execution, producings);
        self.scheduler.schedule();
        return self.delayUntilFinishTask(reducingTaskName);
    })
    .then(function (reducing) {
        return self.receive(reducing.workerName, reducing.task.name);
    })
    .then(function (image) {
        var d = Q.defer();
        var elapsed = (Date.now() | 0) - execution.startTime;
        self.log('Master', 'Elapsed time of execution ' + execution.name + ': ' + elapsed + 'ms');
        d.resolve(image);
        return d.promise;
    });
};

Master.prototype._createIntermediateReducing = function (session, execution, producingTaskPromises) {
    var self = this;
    var d = Q.defer();

    var reducingUnit = Math.sqrt(producingTaskPromises.length) | 0;
    var currentUnit = 0;
    var produceds = [];

    var totalProduced = producingTaskPromises.length;
    var currentProduced = 0;

    var reducingPromises = [];

    var producingFinished = function (producing) {
        ++currentUnit;
        ++currentProduced;

        produceds.push(producing);

        if (currentUnit === reducingUnit || currentProduced === totalProduced) {
            reducingPromises.push(
                    self.delayUntilFinishTask(
                        self.scheduler.createReducingTask(session, execution, produceds)));
            self.scheduler.schedule();
            produceds = [];
            currentUnit = 0;
        }

        if (currentProduced === totalProduced) {
            Q.all(reducingPromises).then(function (reducings) {
                d.resolve(reducings);
            });
        }
    };

    producingTaskPromises.map(function (producingTaskPromise) {
        producingTaskPromise.then(producingFinished);
    });

    return d.promise;
};

//
// Receiving result
//

Master.prototype.receive = function (workerName, taskName) {
    var self = this;
    var d = Q.defer();

    var worker = self.workers[workerName];

    // self.master.log('Master', 'retrieving ...');

    request({
        uri: 'http://' + worker.host + ':' + worker.resourcePort + '/results/' + taskName,
        encoding: null
    }, function (error, response, body) {
        if (error) {
            self.log('Master', error);
            d.reject(error);
        } else {
            // self.master.log('Master', 'retrieving finished!');
            d.resolve(body);
        }
    });

    return d.promise;
};

module.exports = Master;
