CXX = g++

CXXFLAGS = -O2 -Wall -g -std=c++11
LDFLAGS = -lthrift -lthriftcpp2 -lpthread -L/usr/local/lib -lfolly -lglog -lboost_system

TARGET_SERVER = aobench_server
OBJS_SERVER = AobenchConstants.o AobenchTypes.o AobenchServiceServer.o AobenchService.o

TARGET_CLIENT = aobench_client
OBJS_CLIENT = AobenchConstants.o AobenchTypes.o AobenchServiceClient.o AobenchService.o

all: $(TARGET_SERVER) $(TARGET_CLIENT)

$(TARGET_SERVER): $(OBJS_SERVER)
	$(CXX) -o $(TARGET_SERVER) $(OBJS_SERVER) $(LDFLAGS) $(CXXFLAGS)

$(TARGET_CLIENT): $(OBJS_CLIENT)
	$(CXX) -o $(TARGET_CLIENT) $(OBJS_CLIENT) $(LDFLAGS) $(CXXFLAGS)

AobenchServiceServer.o: AobenchServiceServer.cpp
AobenchServiceClient.o: AobenchServiceClient.cpp
AobenchService.o: gen-cpp2/AobenchService.cpp gen-cpp2/AobenchService.h
	$(CXX) $(CXXFLAGS) -c -o $@ $<
AobenchConstants.o: gen-cpp2/aobench_constants.cpp gen-cpp2/aobench_constants.h
	$(CXX) $(CXXFLAGS) -c -o $@ $<

AobenchTypes.o: gen-cpp2/aobench_types.cpp gen-cpp2/aobench_types.h
	$(CXX) $(CXXFLAGS) -c -o $@ $<

clean:
	$(RM) $(TARGET_SERVER) $(OBJS_SERVER)
