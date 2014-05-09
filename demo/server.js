"use strict";

require('shelljs/global');

var Q = require('q');
var http = require('http');
var connect = require('connect');
var express = require('express');

var app = express();
var io = require('socket.io');
var fs = require('fs');
var spawn = require('child_process').spawn;
var redis = require('redis');
var redback = require('redback');
var assert = require('assert');

// sesssion store
var sessionStore = new express.session.MemoryStore();
//var RedisStore = require('connect-redis')(express);
//var sessionStore = new RedisStore();

// Config
var port        = 7000;

//var redisPort       = 16379; // gce
var redisPort       = 6379; // local
//var redisServerAddr = "172.17.0.78"; // redis in docker
//var redisServerAddr = "127.0.0.1"; // redis in local
var redisServerAddr = process.env.REDIS_HOST;

var clang = 'clang';

var globalID = 0;
var sessionToSocketTable = {};

console.log('redisoPort:' + redisPort)

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
  //console.log(req.params.id);
  console.log('req:sessionID:' + req.sessionID);

  var shader_id = parseInt(req.params.id);
  if (shader_id > 0) {
    console.log('store: sess: ' + req.sessionID + ', shaderid: ' + shader_id);
    storeShaderID(req.sessionID, shader_id, function() {
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

//var io = socket_io.listen(app, {secure: true, 'log level': 0});
// Prevent websocket error.
// http://stackoverflow.com/questions/11350279/socket-io-does-not-work-on-firefox-chrome
//io.configure('development', function(){
//  io.set('transports', ['xhr-polling']);
//});

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
 
  // Codes to share session between socket.io and http(express).
  io.set('authorization', function (handshakeData, callback) {
     console.log(handshakeData);
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


// Assume data is already encoded in base64 format.
function generateDataURI(mime, data)
{
  var datauri = 'data:' + mime + ';base64,' + data;
  return datauri;
}

var redisClient    = redis.createClient(redisPort, redisServerAddr);
var redbackClient  = redback.use(redisClient);
var renderCmdQ     = redbackClient.createCappedList('render_cmd', 1);

function storeShaderID(sessionID, shaderID, callback) {
  var key = 'sleditor:shader-id-map';

  redisClient.hset(key, sessionID, shaderID, function(err, reply) {
      assert(!err);
      return callback();
  });
}

function getShaderID(sessionID, callback) {
  var key = 'sleditor:shader-id-map';

  redisClient.hget(key, sessionID, function(err, reply) {
    console.log("session: " + sessionID);
    console.log("val    : " + reply);

    var val = parseInt(reply)

    if (err || reply == undefined || (val < 1)) {
      return callback(-1); // not assigned yet?
    } else {
      return callback(val);
    }

  });
}

function assignShaderID(sessionID, callback) {

  getShaderID(sessionID, function(shader_id) {
    if (shader_id < 1) {

      // Assign new ID
      redisClient.setnx('lte-counter', 1, function(err, reply) {

        console.log('setnx');

        assert(!err);

        redisClient.get('lte-counter', function(err, reply) {

          console.log('reply:' + reply);

          shader_id = parseInt(reply); // store
          assert(shader_id > 0);

          console.log('shader_id:assign: ' + shader_id);

          // count up
          redisClient.incr('lte-counter');

          console.log("shader_id: " + shader_id);
          //socket.emit("session_notify", shader_id);

          storeShaderID(sessionID, shader_id, function() {
            callback(shader_id);
          });

        });
      });

    } else {
      callback(shader_id);        
    }

  });
}



//
// -------------------------------
//
io.sockets.on('connection', function (socket) {

  var shaderID  = socket.handshake.shaderID;
  var sessionID = socket.handshake.sessionID;
  console.log('sessionID: ' + sessionID);
  console.log('shaderID:  ' + shaderID);
  //assert(sessionID);
  
  if (sessionToSocketTable[sessionID] == undefined) {
    sessionToSocketTable[sessionID] = []
  }
  sessionToSocketTable[sessionID].push(socket)
  console.log(sessionID + ' len = ' + sessionToSocketTable[sessionID].length);


  console.log('server: sock io connect');

  socket.emit("session_notify", socket.handshake.shaderID);

  function initShaderCode(socket, id) {

    console.log('initShderCode. id = ' + id);

    assert(id != undefined);
    assert(id > 0);

    try {
      var content = fs.readFileSync(__dirname + "/data/" + id + "/shader.c");
    } catch (err) {
      console.log("file not found: " + id + ". read default shader."); 
      var content = fs.readFileSync(__dirname + "/data/0/shader.c"); // must exist
    }

    socket.emit("init_code", content.toString());
  }

  //// Look-up existing id
  //getShaderID(sessionID, function(id) {

  //  // save state
  //  shader_id = id;

  //  console.log('id: ' + id);

  //  initShaderCode(socket, id);
  //});

  // Also connect to redis channel
  // [node] <- [renderer]
  //var renderNotify = redback.createClient(redisPort, redisServerAddr).createChannel('render_notify').subscribe();

  //renderNotify.on('message', function(msg) {

  //  // grab key
  //  redisClient.get('render_image', function(err, reply) {
  //    if (!err) {
  //      //console.log("====> render_image get");
  //      // reply is JSON string
  //      var jsonJpeg = JSON.parse(reply)

  //      var uri = generateDataURI('image/jpeg', jsonJpeg['jpegdata']);
  //      socket.emit('render_data', uri);
  //    }
  //  });


  //});

  // register ack handler from redis
  function render_ack(session_id) {
    console.log('wait ack: ' + session_id);

    var key = 'lte-ack:' + session_id;
    var timeout = 3600;
    var rc    = redis.createClient(redisPort, redisServerAddr);

    // first clear ack key
    rc.del(key, function(err, reply) {

      rc.blpop(key, timeout, function(err, reply) {
        //console.log(err, reply);

        if (err == null && reply == null) { // maybe timeout
          // do nothing
        } else if (reply && reply[1] == undefined) {
          // do nothing.
        } else if (!err && !reply) {
          //console.log('ack:' + reply[1]);
          //return ok;
        } else {
          console.log('ack:');
          console.log(reply);
         
          if (reply && (reply.length > 1) && (reply[1] != undefined)) {
            var j = JSON.parse(reply[1]);

            console.log('= code =: ' + j['code']);

            if (j['code'] && j['code'] == 'linkerr') {

              var infos = {};
              infos['stderr'] += j['log'];
              infos['results'] = []; // empty
              socket.emit('compile_result', infos);
              console.log("linkerr: " + j['log']);

              // resubmit event
              setTimeout(render_ack(session_id), 1000);

            } else if (j['code'] && j['code'] == 'ok') {

              var image_key = 'render_image:' + session_id;
              console.log('grab image: ' + image_key);

              // grab result
              rc.get(image_key, function(err, reply) {
                if (!err) {

                  console.log("====> render_image get");

                  // reply is JSON string
                  var jsonJpeg = JSON.parse(reply)

                  var uri = generateDataURI('image/jpeg', jsonJpeg['jpegdata']);
                  console.log('emit render_data');

                  //console.log(sessionToSocketTable[session_id]);

                  for (var i in sessionToSocketTable[session_id]) {
                    console.log('gid = ' + sessionToSocketTable[session_id][i].globalID);
                    sessionToSocketTable[session_id][i].emit('render_data', uri);
                  }

                  console.log('emit render_data to done');

                 // resubmit event
                 setTimeout(render_ack(session_id), 1000);
               }
             });
            }
          }
        }

        //console.log('->-- ack end -----');
        //console.log(err, reply);
        //console.log('-<-- ack end -----');
        
      });
    });
  }

  // initShaderCode(socket, id);
  // render_ack(sessionID);
  function myInc() {
    socket.globalID = globalID++;
  }

  Q.fcall(initShaderCode(socket, socket.handshake.shaderID)).then(myInc()).then(render_ack(sessionID));


  socket.on('msg', function(param) {
    // code.
    
    //console.log('msg:' + param['code'])
    var shader_id = socket.handshake.shaderID;
    console.log('shader_id:' + shader_id);

    assert(shader_id > 0); // @fixme

    var dir = __dirname + '/data/' + shader_id
    var filepath = dir + '/shader.c';
    var diag_opt = "-fdiagnostics-print-source-range-info"
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir)
    }
    fs.writeFileSync(filepath, param['code'])

    console.log(filepath)

    var infos = {}
    infos['results'] = []
    infos['stderr'] = ""
    
    var c = spawn(clang, ["-c", "-I/tmp/server", "-D__LTE_CLSHADER__", diag_opt, filepath])

    c.stdout.setEncoding('utf8');
    c.stdout.on('data', function (data) {
      //console.log('stdout: ' + data);
    });

    c.stderr.setEncoding('utf8');
    c.stderr.on('data', function (data) {
      var str = data.toString(), lines = str.split(/(\r?\n)/g);
      infos['stderr'] += str;
      for (var i = 0; i < lines.length; i++) {

        var line = lines[i]
        var reRange = /shader.c:(\d+):(\d+):\{(\d+):(\d+)-(\d+):(\d+)\}.*: (\w.+):(.*)/
        var ret = reRange.exec(line)

        if (ret != null) {
          var rangeinfo =  [ret[3], ret[4], ret[5], ret[6]]
          var ty = ret[7]
          if (ty == 'fatal error') ty = 'error'
          var msg = ret[8]
          var info = {}
          info['range'] = rangeinfo
          info['type'] = ty
          info['message'] = msg
          //console.log(info)
          infos['results'].push(info);
        } else {

          var reLine = /shader.c:(\d+):(\d+): (\w.+):(.*)/

          var ret = reLine.exec(line)
          if (ret != null) {
            var lineinfo = [ret[1], ret[2]]
            var ty = ret[3]
            if (ty == 'fatal error') ty = 'error'
            var msg = ret[4]
            var info = {}
            info['line'] = lineinfo
            info['type'] = ty
            info['message'] = msg
            //console.log(info)
            infos['results'].push(info);
          }

        }

      }

    });

    c.on('close', function(code) {
      console.log('close: ' + code);
      //console.log('infos:' + JSON.stringify(infos));
      socket.emit('compile_result', infos);


      if (code == 0) {

        var task = {session_id: sessionID, shader_id: shader_id, code: param['code']};
        //console.log(task);

        console.log('render-q');
        redisClient.lpush('render-q', JSON.stringify(task));

        //return;

        //// Compile OK. Next do link check.
        //exec("~/work/glrs-branch/bin/lte --linkcheck -c teapot_redis.json > /dev/null", function(linkcheckcode, linkcheckoutput) {
        ////console.log("linkcheck");
        ////exec("./run_docker_lte_linkcheck.sh > /dev/null", function(linkcheckcode, linkcheckoutput) {
        //  if (linkcheckcode != 1) {
        //    console.log("linkerr: " + linkcheckoutput);
        //    infos['stderr'] += linkcheckoutput;
        //    socket.emit('compile_result', infos);
        //  } else {
        //    // Link OK
        //    console.log("linkcheck OK");
        //    
        //    // kick render!
        //    //cmd = { "shader_changed": {} }
        //    //var buffer = new Buffer(JSON.stringify(cmd))
        //    //renderCmdQ.push(buffer, function() {
        //    //  console.log("kick");
        //    //});
        //    
        //    return;

        //    var cmd = "./run_docker_lte.sh"
        //    console.log(cmd)
        //    r = spawn(cmd, [session]);
        //    r.on('close', function(code) {
        //      if (code == 0) {
        //        var image_key = 'render_image';

        //        if (session > 0 ) {
        //          image_key += ':' + session
        //        }

        //        console.log('grab image: ' + image_key);

        //        // grab result
        //        redisClient.get(image_key, function(err, reply) {
        //          if (!err) {

        //            console.log("====> render_image get");

        //            // reply is JSON string
        //            var jsonJpeg = JSON.parse(reply)

        //            var uri = generateDataURI('image/jpeg', jsonJpeg['jpegdata']);
        //            socket.emit('render_data', uri);
        //          }
        //        });

        //      }
        //    });
        //  }
        //});
      }
    });
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

//app.listen(port);

console.log("port:" + port);

function handler (req, res) {
  //console.log(req.url)
  var filename = req.url
  var tmp = req.url.split('.');
  var type = tmp[tmp.length-1];
  //console.log(filename)
  //console.log(type)
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
        //console.log('js:' + req.url)
        res.writeHead(200, {'Content-Type': 'text/javascript'});
        break;
      case 'css':
        res.writeHead(200, {'Content-Type': 'text/css'});
        break;
    }
    res.end(data);
  });
}

