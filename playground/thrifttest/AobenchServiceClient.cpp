#include <cstdio>
#include <cstdlib>
#include <vector>

#include <thrift/lib/cpp2/async/HeaderClientChannel.h>
#include <thrift/lib/cpp2/async/RequestChannel.h>

#include <thrift/lib/cpp/async/TEventBase.h>
#include <thrift/lib/cpp/async/TAsyncSocket.h>
#include <thrift/lib/cpp/async/TAsyncServerSocket.h>

#include <thrift/lib/cpp2/async/StubSaslClient.h>
#include <thrift/lib/cpp2/async/StubSaslServer.h>

#include "gen-cpp2/AobenchService.h"

using namespace ::apache::thrift;
using namespace ::apache::thrift::protocol;
using namespace ::apache::thrift::transport;
using namespace ::apache::thrift::async;

namespace {
void
saveppm(const char *fname, int w, int h, unsigned char *img)
{
    FILE *fp;

    fp = fopen(fname, "wb");
    assert(fp);

    fprintf(fp, "P6\n");
    fprintf(fp, "%d %d\n", w, h);
    fprintf(fp, "255\n");
    fwrite(img, w * h * 3, 1, fp);
    fclose(fp);
}

void saveppm_sum(const char *fname, int w, int h, unsigned char **images, int num) {
  int *result_int = new int [3 * w * h];
  for (int i = 0; i < 3 * w * h; ++i) {
    result_int[i] = 0;
  }
  for (int i = 0; i < num; ++i) {
    for (int j = 0; j < 3 * w * h; ++j) {
      result_int[j] += images[i][j];
    }
  }

  unsigned char* result = new unsigned char [3 * w * h];
  for (int i = 0; i < 3 * w * h; ++i) {
    result[i] = result_int[i] / num;
  }
  delete[] result_int;

  saveppm(fname, w, h, result);

  delete[] result;
}

}  // namespace

int main() {
  TEventBase base;

  const int ports[] = { 9090, 9091, 9092, 9093, 9094 };
  const char *hosts[] = {"127.0.0.1", "127.0.0.1", "127.0.0.1", "127.0.0.1", "127.0.0.1"};
  unsigned char* images[5];
  int cur_pos = 0;

  std::vector<std::shared_ptr<TAsyncSocket>> sockets;
  std::vector<std::shared_ptr<aobench::cpp2::AobenchServiceAsyncClient>> clients;
  for (int i = 0; i < 5; ++i) {
    std::shared_ptr<TAsyncSocket> socket(
      TAsyncSocket::newSocket(&base, hosts[i], ports[i]));
    sockets.push_back(socket);

    auto client_channel = HeaderClientChannel::newChannel(socket);
    auto client = std::make_shared<aobench::cpp2::AobenchServiceAsyncClient>(std::move(client_channel));
    clients.push_back(client);

    client->render(
        [&](ClientReceiveState&& state) {
          std::string result;
          fprintf(stderr, "received\n");
          try {
            aobench::cpp2::AobenchServiceAsyncClient::recv_render(result, state);

            unsigned char* img = new unsigned char [result.size()];
            for (int i = 0; i < static_cast<int>(result.size()); ++i) {
              img[i] = static_cast<unsigned char>(result[i]);
            }
            images[cur_pos] = img;
            ++cur_pos;
            if (cur_pos == 5) {
              saveppm_sum("ao.ppm", 256, 256, images, 5);
              for (int i = 0; i < 5; ++i) {
                delete[] images[i];
              }
              fprintf(stderr, "accumulated\n");
            }
          } catch(const std::exception& ex) {
            fprintf(stderr, "exception thrown %s\n", ex.what());
          }
        }, 256, 256, 2);
  }
  fprintf(stderr, "started\n");
  base.loop();
  fprintf(stderr, "finished\n");
}
