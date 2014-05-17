"use strict";

require('shelljs/global');

var Q = require('q');
var http = require('http');
var connect = require('connect');
var express = require('express');
var request = require('request');

var app = express();
var io = require('socket.io');
var fs = require('fs');
var spawn = require('child_process').spawn;
var assert = require('assert');

var sessionStore = new express.session.MemoryStore();

// Configurations
var port                 = 7000;
var restServerAddr       = process.env.REST_HOST;
var clang                = 'clang';

var globalID             = 0;
var sessionToSocketTable = {};

app.configure(function () {
  app.set('port', process.env.PORT || port);
  app.set('secretKey', 'lte2014shader_koku53surikag1');
  app.set('cookieSessionKey', 'sid');

  app.use("/javascripts", express.static(__dirname + '/javascripts'));
  app.use("/css", express.static(__dirname + '/css'));

  app.use("/manual/js", express.static(__dirname + '/manual/js'));
  app.use("/manual/css", express.static(__dirname + '/manual/css'));

  app.use(express.cookieParser(app.get('secretKey')));
  app.use(express.session({key: app.get('cookieSessionKey'),
                           store : sessionStore,
                           cookie : { maxAge: 3600 * 1000} // 1 hour
                          }));
  app.use(express.bodyParser());
  app.use(express.methodOverride());
});

app.get('/', function(req, res) {
  console.log('req:sessionID:' + req.sessionID);
  res.sendfile(__dirname + '/index.html');
});

app.get('/:id', function(req, res) {
  console.log('req:sessionID:' + req.sessionID);

  var shaderID = parseInt(req.params.id);
  if (shaderID > 0) {
    console.log('store: sess: ' + req.sessionID + ', shaderid: ' + shaderID);
    storeShaderID(req.sessionID, shaderID, function() {
      res.sendfile(__dirname + '/index.html');
    });
  } else {
    res.sendfile(__dirname + '/index.html');
  }
});

app.get('/manual/index.html', function(req, res) {
  res.sendfile(__dirname + '/manual/index.html');
});

io = io.listen(http.createServer(app).listen(app.get('port')), function() {
      console.log("Express server & socket.io listening on port " + app.get('port'));
});

io.configure(function(){

  io.enable('browser client minification');  // send minified client
  io.enable('browser client etag');          // apply etag caching logic based on version number
  io.enable('browser client gzip');          // gzip the file

  io.set('log level', 1);

  io.set('transports', [
      'xhr-polling',
      'websocket',
      'flashsocket',
      'htmlfile',
      'jsonp-polling'
  ]);
 
  // Sharing session between socket.io and http(express).
  io.set('authorization', function (handshakeData, callback) {
     console.log('cookie:' + handshakeData.headers.cookie);

     if(handshakeData.headers.cookie) {
        var cookie = require('cookie').parse(decodeURIComponent(handshakeData.headers.cookie));

        // decript cookie
        cookie = connect.utils.parseSignedCookies(cookie, app.get('secretKey'))
        var sessionID = cookie[app.get('cookieSessionKey')];

        console.log('sessionID: ' + sessionID);

        sessionStore.get(sessionID, function(err, session) {
          if (err) {
            console.dir(err);
            return callback(err.message, false);
          } else if (!session) {
            console.log('ERR: session not found in DB.');
            return callback('session not found', false);
          } else {
            console.log('auth ok!');

            assignShaderID(sessionID, function(shaderID) {
              // socket.io can see session data.
              handshakeData.cookie = cookie;
              handshakeData.shaderID  = shaderID;
              handshakeData.sessionID = sessionID;
              handshakeData.sessionStore = sessionStore;
              handshakeData.session = new express.session.Session(handshakeData, session);

              return callback(null, true); // OK
            });
          }
        });
     } else {
        return callback('Cookie must be enabled', false);
     }
  });
});

function generateDataURI(mime, data) {
  return 'data:' + mime + ';base64,' + data.toString('base64');
}

var shaderIDMap = {};

function storeShaderID(sessionID, shaderID, callback) {
  shaderIDMap[sessionID] = shaderID;
  return callback();
}

function getShaderID(sessionID, callback) {
  if (shaderIDMap[sessionID] == undefined) {
    return callback(-1);
  } else {
    return callback(shaderIDMap[sessionID]);
  }
}

function putResourcesWithRest(restSessionID) {
  var resources = [
    {from: "teapot_redis.json",    to: "scene/teapot_redis.json"},
    {from: "teapot_scene.json",    to: "scene/teapot_scene.json"},
    {from: "teapot.material.json", to: "scene/teapot.material.json"},
    {from: "shaders.json",         to: "scene/shaders.json"},
    {from: "teapot.mesh",          to: "scene/teapot.mesh"},
    {from: "shader.c",             to: "shader.c"},
    {from: "shader.h",             to: "shader.h"},
    {from: "procedural-noise.c",   to: "procedural-noise.c"},
    {from: "light.h",              to: "light.h"}
  ];

  var chain = Q.when(0);

  for (var i = 0; i < resources.length; ++i) {
    request.put({
      url: 'http://' + restServerAddr + '/sessions/' + restSessionID + '/resources/' + resources[i]['to'],
      body: fs.readFileSync(__dirname + '/' + resources[i]['from'])
    });
  }

  return;
}

function assignShaderID(sessionID, callback) {
  getShaderID(sessionID, function(shaderID) {
    if (shaderID < 1) {
      request.post({
        url: 'http://' + restServerAddr + '/sessions',
        json: { InputJson: 'scene/teapot_redis.json' }
      }, function(err, res, body) {
        console.log(body);
        var restSessionID = body['SessionId'];
        putResourcesWithRest(restSessionID);
        return callback(restSessionID);
      });
    }
  });
}

// Send custom shader.c to the REST API and render image
function renderWithCustomShader(sessionID, restSessionID, socket, sessionToSocketTable, code) {
  request.put({
    url: 'http://' + restServerAddr + '/sessions/' + restSessionID + '/resources/shader.c',
    body: code
  }, function(err, res, body) {
    request.post({
      url: 'http://' + restServerAddr + '/sessions/' + restSessionID + '/renders',
      encoding: null
    }, function(err, res, body) {
      switch (res.headers['content-type']) {
        case 'application/json':
          var parsed = JSON.parse(body.toString('utf8'));

          console.log('emit compile_result');
          socket.emit('compile_result', {stderr: parsed['Log'], results: []});

          break;
        case 'image/jpeg':
          var dataURI = generateDataURI('image/jpeg', body);

          for (var i = 0; i < sessionToSocketTable[sessionID].length; ++i) {
            sessionToSocketTable[sessionID][i].emit('render_data', dataURI);
          }

          console.log('emit render_data to done');

          break;
        }
    });
  });

  return;
}

// Init local shader code for local clang checking
function initShaderCode(socket, id) {
  console.log('initShderCode: id = ' + id);

  assert(id != undefined);
  assert(id > 0);

  try {
    var content = fs.readFileSync(__dirname + "/data/" + id + "/shader.c");
  } catch (err) {
    console.log("file not found: " + id + "; read default shader"); 
    var content = fs.readFileSync(__dirname + "/data/0/shader.c"); // must exist
  }

  socket.emit("init_code", content.toString());
}

// Do checking with local clang for better error diagnose
function checkWithClang(sessionID, shaderID, socket, code, callback) {
  var dir      = __dirname + '/data/' + shaderID;
  var filepath = dir + '/shader.c';
  var diag_opt = "-fdiagnostics-print-source-range-info";
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir);
  }

  fs.writeFileSync(filepath, code);

  console.log(filepath);

  var infos = {};
  infos['results'] = [];
  infos['stderr'] = "";
  
  var c = spawn(clang, ["-c", "-I/tmp/server", "-D__LTE_CLSHADER__", diag_opt, filepath]);

  c.stdout.setEncoding('utf8');
  c.stderr.setEncoding('utf8');

  c.stderr.on('data', function (data) {
    var str = data.toString(), lines = str.split(/(\r?\n)/g);
    infos['stderr'] += str;
    for (var i = 0; i < lines.length; i++) {
      var line = lines[i]
      var reRange = /shader.c:(\d+):(\d+):\{(\d+):(\d+)-(\d+):(\d+)\}.*: (\w.+):(.*)/
      var ret = reRange.exec(line)
      if (ret != null) {
        infos['results'].push({
          range:   [ret[3], ret[4], ret[5], ret[6]],
          type:    (ret[7] == 'fatal error' ? 'error' : ret[7]),
          message: ret[8] 
        });
      } else {
        var reLine = /shader.c:(\d+):(\d+): (\w.+):(.*)/
        var ret = reLine.exec(line);
        if (ret != null) {
          infos['results'].push({
            line:    [ret[1], ret[2]],
            type:    (ret[3] == 'fatal error' ? 'error' : ret[3]),
            message: ret[4]
          });
        }
      }
    }
  });

  c.on('close', function(code) {
    console.log('close: ' + code);
    socket.emit('compile_result', infos);
    if (code == 0) {
      callback();
    }
  });
}

io.sockets.on('connection', function (socket) {
  var shaderID  = socket.handshake.shaderID;
  var sessionID = socket.handshake.sessionID;
  console.log('sessionID: ' + sessionID);
  console.log('shaderID: ' + shaderID);
  
  if (sessionToSocketTable[sessionID] == undefined) {
    sessionToSocketTable[sessionID] = [];
  }
  sessionToSocketTable[sessionID].push(socket);
  console.log(sessionID + ' len = ' + sessionToSocketTable[sessionID].length);

  console.log('server: sock io connect');

  socket.emit("session_notify", socket.handshake.shaderID);

  Q.fcall(initShaderCode(socket, socket.handshake.shaderID)).then(function() { socket.globalID = globalID++; });

  socket.on('msg', function(param) {
    var shaderID = socket.handshake.shaderID;
    console.log('shaderID:' + shaderID);

    assert(shaderID > 0); // @fixme

    // comment out checkWithClang to avoid checking
    //checkWithClang(sessionID, shaderID, socket, param['code'], function() {
      renderWithCustomShader(sessionID, shaderID, socket, sessionToSocketTable, param['code']);
    //});
  });

  socket.on('error', function(err) {
    console.log('===> socket.io ERR: ' + err);
  });

  socket.on('disconnect', function() {
    if (sessionToSocketTable[socket.sessionID] != undefined) {
      var i = sessionToSocketTable[socket.sessionID].indexOf(socket);
      if (i >= 0) {
        delete sessionToSocketTable[socket.sessionID][i];
      }
      console.log('i = ' + i);
    }
  });
});

console.log("port:" + port);

function handler(req, res) {
  var filename = req.url
  var tmp = req.url.split('.');
  var type = tmp[tmp.length-1];
  if (req.url.match(/^\/(\d+)/)) {
    var re = req.url.match(/^\/(\d+)/);
    var num = parseInt(re[1]);

    console.log("instance: " + num);
    filename = '/index.html'
    type = 'html'
    session = num;

    if (num == 0) {
      sesssion = -1;
    }

  } else if (req.url == '/') {
    console.log('toplevel');
    filename = '/index.html'
    type = 'html'
    session = -1;
  }
  fs.readFile(__dirname + filename,
  function (err, data) {
    if (err) {
      res.writeHead(500);
      return res.end('Error loading ' + filename);
    }
    switch (type) {
      case 'html':
        res.writeHead(200, {'Content-Type': 'text/html'});
        break;
      case 'js':
        res.writeHead(200, {'Content-Type': 'text/javascript'});
        break;
      case 'css':
        res.writeHead(200, {'Content-Type': 'text/css'});
        break;
    }
    res.end(data);
  });
}

