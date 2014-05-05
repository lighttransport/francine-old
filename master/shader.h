//
// Copyright (C) 2012-2014 Light Transport Entertainment Inc.
//
#ifndef __LTE_SHADER_H__
#define __LTE_SHADER_H__

#ifdef __cplusplus
extern "C" {
#endif

#include "light.h"

#ifdef __clang__
#define PACKED __attribute__((packed))
typedef float vec2 __attribute__((ext_vector_type(2)));
typedef float vec3 __attribute__((ext_vector_type(3)));
typedef float vec4 __attribute__((ext_vector_type(4)));
#elif __GNUC__
#define PACKED __attribute__((packed))
//typedef float vec4[4];
typedef float v4sf __attribute__ ((vector_size(16)));

union vec4
{
	v4sf	vec;
	float	f[4];
};
#else
#define PACKED
union vec4
{
	float	f[4];
};
#endif


#ifdef _MSV_VER
#pragma pack(1)
#endif
typedef struct PACKED {
    vec4 Ci;
    vec4 Oi;
    vec4 Ni;            // normal output
    vec4 Cs;
    vec4 Os;
    vec4 I;
    vec4 N;
    vec4 Ng;
    vec4 tangent;
    vec4 binormal;
    vec4 E;
    vec4 P;
    vec4 L;             // Light dir.
    vec4 Lp;            // Light pos. @fixme.
    vec4 sundir;        // Sun direction. @fixme.
    vec4 suncol;        // Sun RGB color. @fixme.
    vec4 Cl;
    vec4 Ol;

    vec4 attributes[8];             // User defined vertex attributes.

    vec4 ambient;
    vec4 diffuse;
    vec4 reflection;
    vec4 refraction;

    float diffuse_uvMat[4];         // 2D matrix
    float reflection_uvMat[4];      // 2D matrix
    float refraction_uvMat[4];      // 2D matrix
    float bump_uvMat[4];            // 2D matrix

    float u, v;
    float s, t;
    int x, y, z, w;

    // material
    int _fixme_has_true_materials;
    float specularity;
    float ior;
    int   fresnel_reflection;

    unsigned int   diffuse_texture_id;
    unsigned int   reflection_texture_id;
    unsigned int   refraction_texture_id;
    unsigned int   bump_texture_id;
    unsigned int   normal_texture_id;

    unsigned int   texture_ids[8]; // User defined texture slot.
    
    float depth;        // in world coord.
    float texture_gamma;

    // Face ID
    unsigned int face_id;       // -1 if no hit.

    float bumpness;

    // --

    float reflection_glossiness;
    float refraction_glossiness;

    // raytrace
    int trace_depth;
    int hit_light_obj;

    // --

    //// internal
    void *shader_engine;

    // material ID
    unsigned short material_id; // MUST BE ushort16
    unsigned short pad0;        // padding.

    /// Temporal
    int picked_matid;

} ShaderEnv;

// builtin-functions
#ifdef __LTE_CLSHADER__
#define LTE_EXTERN
#pragma OPENCL EXTENSION cl_khr_fp64 : enable

#if __has_attribute(__overloadable__)
#  define LTE_CL_OVERLOADABLE __attribute__((__overloadable__))
#else
#  define LTE_CL_OVERLOADABLE
#endif

// Work around for name conflict with math.h

#define FLT_DIG        6
#define FLT_MANT_DIG   24
#define FLT_MAX_10_EXP +38
#define FLT_MAX_EXP    +128
#define FLT_MIN_10_EXP -37
#define FLT_MIN_EXP    -125
#define FLT_RADIX      2
#define FLT_MAX        0x1.fffffep127f
#define FLT_MIN        0x1.0p-126f
#define FLT_EPSILON    0x1.0p-23f

#define M_E_F        2.71828182845904523536028747135f
#define M_LOG2E_F    1.44269504088896340735992468100f
#define M_LOG10E_F   0.434294481903251827651128918917f
#define M_LN2_F      0.693147180559945309417232121458f
#define M_LN10_F     2.30258509299404568401799145468f
#define M_PI_F       3.14159265358979323846264338328f
#define M_PI_2_F     1.57079632679489661923132169164f
#define M_PI_4_F     0.785398163397448309615660845820f
#define M_1_PI_F     0.318309886183790671537767526745f
#define M_2_PI_F     0.636619772367581343075535053490f
#define M_2_SQRTPI_F 1.12837916709551257389615890312f
#define M_SQRT2_F    1.41421356237309504880168872421f
#define M_SQRT1_2_F  0.707106781186547524400844362105f

#define DBL_DIG        15
#define DBL_MANT_DIG   53
#define DBL_MAX_10_EXP +308
#define DBL_MAX_EXP    +1024
#define DBL_MIN_10_EXP -307
#define DBL_MIN_EXP    -1021
#define DBL_MAX        0x1.fffffffffffffp1023
#define DBL_MIN        0x1.0p-1022
#define DBL_EPSILON    0x1.0p-52

#define M_E        2.71828182845904523536028747135
#define M_LOG2E    1.44269504088896340735992468100
#define M_LOG10E   0.434294481903251827651128918917
#define M_LN2      0.693147180559945309417232121458
#define M_LN10     2.30258509299404568401799145468
#define M_PI       3.14159265358979323846264338328
#define M_PI_2     1.57079632679489661923132169164
#define M_PI_4     0.785398163397448309615660845820
#define M_1_PI     0.318309886183790671537767526745
#define M_2_PI     0.636619772367581343075535053490
#define M_2_SQRTPI 1.12837916709551257389615890312
#define M_SQRT2    1.41421356237309504880168872421
#define M_SQRT1_2  0.707106781186547524400844362105

float fmaf(float,float, float);
double fmad(double, double, double);
float fmodf(float,float);
double fmodd(double, double);
float sqrtf(float);
double sqrtd(double);
float fabsf(float);
double fabsd(double);
float acosf(float);
double acosd(double);
float acoshf(float);
double acoshd(double);
float asinf(float);
double asind(double);
float asinhf(float);
double asinhd(double);
float atanf(float);
double atand(double);
float atan2f(float, float);
double atan2d(double, double);
float atanhf(float);
double atanhd(double);
float sinf(float);
double sind(double);
float cosf(float);
double cosd(double);
float tanf(float);
double tand(double);
float expf(float);
double expd(double);
float exp2f(float);
double exp2d(double);
float logf(float);
double logd(double);
float log2f(float);
double log2d(double);
float floorf(float);
double floord(double);
float ceilf(float);
double ceild(double);
float powf(float, float);
double powd(double, double);
float modff(float, float*);
double modfd(double, double*);

int isinff(float);
int isinfd(double);
int isfinitef(float);
int isfinited(double);

#define fma    __cl_fma
#define fmod   __cl_fmod
#define acos   __cl_acos
#define acosh  __cl_acosh
#define asin   __cl_asin
#define asinh  __cl_asinh
#define atan   __cl_atan
#define atan2  __cl_atan2
#define atanh  __cl_atanh
#define fabs   __cl_fabs
#define sqrt   __cl_sqrt
#define ceil   __cl_ceil
#define floor  __cl_floor
#define sin    __cl_sin
#define cos    __cl_cos
#define tan    __cl_tan
#define pow    __cl_pow
#define exp    __cl_exp
#define log    __cl_log
#define exp2   __cl_exp2
#define log2   __cl_log2


#define fract  __cl_fract

#define isfinite  __cl_isfinite
#define isinf     __cl_isinf
#define isnan     __cl_isnan

#define min    __cl_min
#define max    __cl_max

#define sign        __cl_sign
#define radians     __cl_radians
#define degrees     __cl_degrees
#define mix         __cl_mix
#define clamp       __cl_clamp
#define step        __cl_step
#define smoothstep  __cl_smoothstep

#define length    __cl_length
#define distance  __cl_distance
#define dot       __cl_dot
#define cross     __cl_cross
#define normalize __cl_normalize

inline float  LTE_CL_OVERLOADABLE __cl_fma(float x, float y, float z) { return fmaf(x, y, z); }
inline double LTE_CL_OVERLOADABLE __cl_fma(double x, double y, double z) { return fmad(x, y, z); }

inline float  LTE_CL_OVERLOADABLE __cl_fmod(float x, float y) { return fmodf(x, y); }
inline double LTE_CL_OVERLOADABLE __cl_fmod(double x, double y) { return fmodd(x, y); }
inline vec2   LTE_CL_OVERLOADABLE __cl_fmod(vec2 x, vec2 y) {
  vec2 ret;
  ret.x = fmodf(x.x, y.x);
  ret.y = fmodf(x.y, y.y);
  return ret;
}
inline vec3   LTE_CL_OVERLOADABLE __cl_fmod(vec3 x, vec3 y) {
  vec3 ret;
  ret.x = fmodf(x.x, y.x);
  ret.y = fmodf(x.y, y.y);
  ret.z = fmodf(x.z, y.z);
  return ret;
}
inline vec4   LTE_CL_OVERLOADABLE __cl_fmod(vec4 x, vec4 y) {
  vec4 ret;
  ret.x = fmodf(x.x, y.x);
  ret.y = fmodf(x.y, y.y);
  ret.z = fmodf(x.z, y.z);
  ret.w = fmodf(x.w, y.w);
  return ret;
}

inline float  LTE_CL_OVERLOADABLE __cl_acos(float x) { return acosf(x); }
inline double LTE_CL_OVERLOADABLE __cl_acos(double x) { return acosd(x); }
inline vec2 LTE_CL_OVERLOADABLE __cl_acos(vec2 x){
    vec2 ret;
    ret.x = acosf(x.x);
    ret.y = acosf(x.y);
    return ret;
}
inline vec3 LTE_CL_OVERLOADABLE __cl_acos(vec3 x){
    vec3 ret;
    ret.x = acosf(x.x);
    ret.y = acosf(x.y);
    ret.z = acosf(x.z);
    return ret;
}
inline vec4 LTE_CL_OVERLOADABLE __cl_acos(vec4 x){
    vec4 ret;
    ret.x = acosf(x.x);
    ret.y = acosf(x.y);
    ret.z = acosf(x.z);
    ret.w = acosf(x.w);
    return ret;
}

inline float  LTE_CL_OVERLOADABLE __cl_acosh(float x) { return acoshf(x); }
inline double LTE_CL_OVERLOADABLE __cl_acosh(double x) { return acoshd(x); }

inline float  LTE_CL_OVERLOADABLE __cl_asin(float x) { return asinf(x); }
inline double LTE_CL_OVERLOADABLE __cl_asin(double x) { return asind(x); }
inline vec2 LTE_CL_OVERLOADABLE __cl_asin(vec2 x){
    vec2 ret;
    ret.x = asinf(x.x);
    ret.y = asinf(x.y);
    return ret;
}
inline vec3 LTE_CL_OVERLOADABLE __cl_asin(vec3 x){
    vec3 ret;
    ret.x = asinf(x.x);
    ret.y = asinf(x.y);
    ret.z = asinf(x.z);
    return ret;
}
inline vec4 LTE_CL_OVERLOADABLE __cl_asin(vec4 x){
    vec4 ret;
    ret.x = asinf(x.x);
    ret.y = asinf(x.y);
    ret.z = asinf(x.z);
    ret.w = asinf(x.w);
    return ret;
}

inline float  LTE_CL_OVERLOADABLE __cl_asinh(float x) { return asinhf(x); }
inline double LTE_CL_OVERLOADABLE __cl_asinh(double x) { return asinhd(x); }

inline float  LTE_CL_OVERLOADABLE __cl_atan(float x) { return atanf(x); }
inline double LTE_CL_OVERLOADABLE __cl_atan(double x) { return atand(x); }
inline vec2 LTE_CL_OVERLOADABLE __cl_atan(vec2 x){
    vec2 ret;
    ret.x = atanf(x.x);
    ret.y = atanf(x.y);
    return ret;
}
inline vec3 LTE_CL_OVERLOADABLE __cl_atan(vec3 x){
    vec3 ret;
    ret.x = atanf(x.x);
    ret.y = atanf(x.y);
    ret.z = atanf(x.z);
    return ret;
}
inline vec4 LTE_CL_OVERLOADABLE __cl_atan(vec4 x){
    vec4 ret;
    ret.x = atanf(x.x);
    ret.y = atanf(x.y);
    ret.z = atanf(x.z);
    ret.w = atanf(x.w);
    return ret;
}

inline float  LTE_CL_OVERLOADABLE __cl_atan2(float x, float y) { return atan2f(x, y); }
inline double LTE_CL_OVERLOADABLE __cl_atan2(double x, float y) { return atan2d(x, y); }
inline vec2 LTE_CL_OVERLOADABLE __cl_atan2(vec2 x, vec2 y){
    vec2 ret;
    ret.x = atan2f(x.x, y.x);
    ret.y = atan2f(x.y, y.y);
    return ret;
}
inline vec3 LTE_CL_OVERLOADABLE __cl_atan2(vec3 x, vec3 y){
    vec3 ret;
    ret.x = atan2f(x.x, y.x);
    ret.y = atan2f(x.y, y.y);
    ret.z = atan2f(x.z, y.z);
    return ret;
}
inline vec4 LTE_CL_OVERLOADABLE __cl_atan2(vec4 x, vec4 y){
    vec4 ret;
    ret.x = atan2f(x.x, y.x);
    ret.y = atan2f(x.y, y.y);
    ret.z = atan2f(x.z, y.z);
    ret.w = atan2f(x.w, y.w);
    return ret;
}

inline float  LTE_CL_OVERLOADABLE __cl_atanh(float x) { return atanhf(x); }
inline double LTE_CL_OVERLOADABLE __cl_atanh(double x) { return atanhd(x); }

inline float  LTE_CL_OVERLOADABLE __cl_fabs(float x) { return fabsf(x); }
inline double LTE_CL_OVERLOADABLE __cl_fabs(double x) { return fabsd(x); }
inline vec2   LTE_CL_OVERLOADABLE __cl_fabs(vec2 x) {
  vec2 ret;
  ret.x = fabsf(x.x);
  ret.y = fabsf(x.y);
  return ret;
}
inline vec3   LTE_CL_OVERLOADABLE __cl_fabs(vec3 x) {
  vec3 ret;
  ret.x = fabsf(x.x);
  ret.y = fabsf(x.y);
  ret.z = fabsf(x.z);
  return ret;
}
inline vec4   LTE_CL_OVERLOADABLE __cl_fabs(vec4 x) {
  vec4 ret;
  ret.x = fabsf(x.x);
  ret.y = fabsf(x.y);
  ret.z = fabsf(x.z);
  ret.w = fabsf(x.w);
  return ret;
}

inline float  LTE_CL_OVERLOADABLE __cl_sqrt(float x) { return sqrtf(x); }
inline double LTE_CL_OVERLOADABLE __cl_sqrt(double x) { return sqrtd(x); }
inline vec2 LTE_CL_OVERLOADABLE __cl_sqrt(vec2 x){
    vec2 ret;
    ret.x = sqrtf(x.x);
    ret.y = sqrtf(x.y);
    return ret;
}
inline vec3 LTE_CL_OVERLOADABLE __cl_sqrt(vec3 x){
    vec3 ret;
    ret.x = sqrtf(x.x);
    ret.y = sqrtf(x.y);
    ret.z = sqrtf(x.z);
    return ret;
}
inline vec4 LTE_CL_OVERLOADABLE __cl_sqrt(vec4 x){
    vec4 ret;
    ret.x = sqrtf(x.x);
    ret.y = sqrtf(x.y);
    ret.z = sqrtf(x.z);
    ret.w = sqrtf(x.w);
    return ret;
}

inline float  LTE_CL_OVERLOADABLE __cl_sin(float x) { return sinf(x); }
inline double LTE_CL_OVERLOADABLE __cl_sin(double x) { return sind(x); }
inline vec2 LTE_CL_OVERLOADABLE __cl_sin(vec2 x){
    vec2 ret;
    ret.x = sinf(x.x);
    ret.y = sinf(x.y);
    return ret;
}
inline vec3 LTE_CL_OVERLOADABLE __cl_sin(vec3 x){
    vec3 ret;
    ret.x = sinf(x.x);
    ret.y = sinf(x.y);
    ret.z = sinf(x.z);
    return ret;
}
inline vec4 LTE_CL_OVERLOADABLE __cl_sin(vec4 x){
    vec4 ret;
    ret.x = sinf(x.x);
    ret.y = sinf(x.y);
    ret.z = sinf(x.z);
    ret.w = sinf(x.w);
    return ret;
}

inline float  LTE_CL_OVERLOADABLE __cl_cos(float x) { return cosf(x); }
inline double LTE_CL_OVERLOADABLE __cl_cos(double x) { return cosd(x); }
inline vec2 LTE_CL_OVERLOADABLE __cl_cos(vec2 x){
    vec2 ret;
    ret.x = cosf(x.x);
    ret.y = cosf(x.y);
    return ret;
}
inline vec3 LTE_CL_OVERLOADABLE __cl_cos(vec3 x){
    vec3 ret;
    ret.x = cosf(x.x);
    ret.y = cosf(x.y);
    ret.z = cosf(x.z);
    return ret;
}
inline vec4 LTE_CL_OVERLOADABLE __cl_cos(vec4 x){
    vec4 ret;
    ret.x = cosf(x.x);
    ret.y = cosf(x.y);
    ret.z = cosf(x.z);
    ret.w = cosf(x.w);
    return ret;
}

inline float  LTE_CL_OVERLOADABLE __cl_min(float x, float y) { return (x < y) ? x : y; }
inline double LTE_CL_OVERLOADABLE __cl_min(double x, float y) { return (x < y) ? x : y; }
inline vec2  LTE_CL_OVERLOADABLE __cl_min(vec2 x, vec2 y) {
  vec2 ret;
  ret.x = (x.x < y.x) ? x.x : y.x;
  ret.y = (x.y < y.y) ? x.y : y.y;
  return ret;
}
inline vec3  LTE_CL_OVERLOADABLE __cl_min(vec3 x, vec3 y) {
  vec3 ret;
  ret.x = (x.x < y.x) ? x.x : y.x;
  ret.y = (x.y < y.y) ? x.y : y.y;
  ret.z = (x.z < y.z) ? x.z : y.z;
  return ret;
}
inline vec4  LTE_CL_OVERLOADABLE __cl_min(vec4 x, vec4 y) {
  vec4 ret;
  ret.x = (x.x < y.x) ? x.x : y.x;
  ret.y = (x.y < y.y) ? x.y : y.y;
  ret.z = (x.z < y.z) ? x.z : y.z;
  ret.w = (x.w < y.w) ? x.w : y.w;
  return ret;
}

inline float  LTE_CL_OVERLOADABLE __cl_max(float x, float y) { return (x > y) ? x : y; }
inline double LTE_CL_OVERLOADABLE __cl_max(double x, float y ) { return (x > y) ? x : y; }
inline vec2  LTE_CL_OVERLOADABLE __cl_max(vec2 x, vec2 y) {
  vec2 ret;
  ret.x = (x.x > y.x) ? x.x : y.x;
  ret.y = (x.y > y.y) ? x.y : y.y;
  return ret;
}
inline vec3  LTE_CL_OVERLOADABLE __cl_max(vec3 x, vec3 y) {
  vec3 ret;
  ret.x = (x.x > y.x) ? x.x : y.x;
  ret.y = (x.y > y.y) ? x.y : y.y;
  ret.z = (x.z > y.z) ? x.z : y.z;
  return ret;
}
inline vec4  LTE_CL_OVERLOADABLE __cl_max(vec4 x, vec4 y) {
  vec4 ret;
  ret.x = (x.x > y.x) ? x.x : y.x;
  ret.y = (x.y > y.y) ? x.y : y.y;
  ret.z = (x.z > y.z) ? x.z : y.z;
  ret.w = (x.w > y.w) ? x.w : y.w;
  return ret;
}

inline float  LTE_CL_OVERLOADABLE __cl_tan(float x) { return tanf(x); }
inline double LTE_CL_OVERLOADABLE __cl_tan(double x) { return tand(x); }
inline vec2 LTE_CL_OVERLOADABLE __cl_tan(vec2 x){
    vec2 ret;
    ret.x = tanf(x.x);
    ret.y = tanf(x.y);
    return ret;
}
inline vec3 LTE_CL_OVERLOADABLE __cl_tan(vec3 x){
    vec3 ret;
    ret.x = tanf(x.x);
    ret.y = tanf(x.y);
    ret.z = tanf(x.z);
    return ret;
}
inline vec4 LTE_CL_OVERLOADABLE __cl_tan(vec4 x){
    vec4 ret;
    ret.x = tanf(x.x);
    ret.y = tanf(x.y);
    ret.z = tanf(x.z);
    ret.w = tanf(x.w);
    return ret;
}

inline float  LTE_CL_OVERLOADABLE __cl_ceil(float x) { return ceilf(x); }
inline double LTE_CL_OVERLOADABLE __cl_ceil(double x) { return ceild(x); }
inline vec2 LTE_CL_OVERLOADABLE __cl_ceil(vec2 x){
    vec2 ret;
    ret.x = ceild(x.x);
    ret.y = ceild(x.y);
    return ret;
}
inline vec3 LTE_CL_OVERLOADABLE __cl_ceil(vec3 x){
    vec3 ret;
    ret.x = ceild(x.x);
    ret.y = ceild(x.y);
    ret.z = ceild(x.z);
    return ret;
}
inline vec4 LTE_CL_OVERLOADABLE __cl_ceil(vec4 x){
    vec4 ret;
    ret.x = ceild(x.x);
    ret.y = ceild(x.y);
    ret.z = ceild(x.z);
    ret.w = ceild(x.w);
    return ret;
}

inline float  LTE_CL_OVERLOADABLE __cl_floor(float x) { return floorf(x); }
inline double LTE_CL_OVERLOADABLE __cl_floor(double x) { return floord(x); }
inline vec2  LTE_CL_OVERLOADABLE __cl_floor(vec2 x) {
 vec2 ret;
 ret.x = floorf(x.x);
 ret.y = floorf(x.y);
 return ret;
}
inline vec3  LTE_CL_OVERLOADABLE __cl_floor(vec3 x) {
 vec3 ret;
 ret.x = floorf(x.x);
 ret.y = floorf(x.y);
 ret.z = floorf(x.z);
 return ret;
}
inline vec4  LTE_CL_OVERLOADABLE __cl_floor(vec4 x) {
 vec4 ret;
 ret.x = floorf(x.x);
 ret.y = floorf(x.y);
 ret.z = floorf(x.z);
 ret.w = floorf(x.w);
 return ret;
}

inline float  LTE_CL_OVERLOADABLE __cl_exp(float x) { return expf(x); }
inline double LTE_CL_OVERLOADABLE __cl_exp(double x) { return expd(x); }
inline vec2 LTE_CL_OVERLOADABLE __cl_exp(vec2 x){
    vec2 ret;
    ret.x = expf(x.x);
    ret.y = expf(x.y);
    return ret;
}
inline vec3 LTE_CL_OVERLOADABLE __cl_exp(vec3 x){
    vec3 ret;
    ret.x = expf(x.x);
    ret.y = expf(x.y);
    ret.z = expf(x.z);
    return ret;
}
inline vec4 LTE_CL_OVERLOADABLE __cl_exp(vec4 x){
    vec4 ret;
    ret.x = expf(x.x);
    ret.y = expf(x.y);
    ret.z = expf(x.z);
    ret.w = expf(x.w);
    return ret;
}

inline float  LTE_CL_OVERLOADABLE __cl_exp2(float x) { return exp2f(x); }
inline double LTE_CL_OVERLOADABLE __cl_exp2(double x) { return exp2d(x); }
inline vec2 LTE_CL_OVERLOADABLE __cl_exp2(vec2 x){
    vec2 ret;
    ret.x = exp2f(x.x);
    ret.y = exp2f(x.y);
    return ret;
}
inline vec3 LTE_CL_OVERLOADABLE __cl_exp2(vec3 x){
    vec3 ret;
    ret.x = exp2f(x.x);
    ret.y = exp2f(x.y);
    ret.z = exp2f(x.z);
    return ret;
}
inline vec4 LTE_CL_OVERLOADABLE __cl_exp2(vec4 x){
    vec4 ret;
    ret.x = exp2f(x.x);
    ret.y = exp2f(x.y);
    ret.z = exp2f(x.z);
    ret.w = exp2f(x.w);
    return ret;
}

inline float  LTE_CL_OVERLOADABLE __cl_log(float x) { return logf(x); }
inline double LTE_CL_OVERLOADABLE __cl_log(double x) { return logd(x); }
inline vec2 LTE_CL_OVERLOADABLE __cl_log(vec2 x){
    vec2 ret;
    ret.x = logf(x.x);
    ret.y = logf(x.y);
    return ret;
}
inline vec3 LTE_CL_OVERLOADABLE __cl_log(vec3 x){
    vec3 ret;
    ret.x = logf(x.x);
    ret.y = logf(x.y);
    ret.z = logf(x.z);
    return ret;
}
inline vec4 LTE_CL_OVERLOADABLE __cl_log(vec4 x){
    vec4 ret;
    ret.x = logf(x.x);
    ret.y = logf(x.y);
    ret.z = logf(x.z);
    ret.w = logf(x.w);
    return ret;
}

inline float  LTE_CL_OVERLOADABLE __cl_log2(float x) { return log2f(x); }
inline double LTE_CL_OVERLOADABLE __cl_log2(double x) { return log2d(x); }
inline vec2 LTE_CL_OVERLOADABLE __cl_log2(vec2 x){
    vec2 ret;
    ret.x = log2f(x.x);
    ret.y = log2f(x.y);
    return ret;
}
inline vec3 LTE_CL_OVERLOADABLE __cl_log2(vec3 x){
    vec3 ret;
    ret.x = log2f(x.x);
    ret.y = log2f(x.y);
    ret.z = log2f(x.z);
    return ret;
}
inline vec4 LTE_CL_OVERLOADABLE __cl_log2(vec4 x){
    vec4 ret;
    ret.x = log2f(x.x);
    ret.y = log2f(x.y);
    ret.z = log2f(x.z);
    ret.w = log2f(x.w);
    return ret;
}

inline float  LTE_CL_OVERLOADABLE __cl_pow(float x, float y) { return powf(x, y); }
inline double LTE_CL_OVERLOADABLE __cl_pow(double x, double y) { return powd(x, y); }
inline vec2 LTE_CL_OVERLOADABLE __cl_pow(vec2 x, vec2 y){
    vec2 ret;
    ret.x = powf(x.x, y.x);
    ret.y = powf(x.y, y.y);
    return ret;
}
inline vec3 LTE_CL_OVERLOADABLE __cl_pow(vec3 x, vec3 y){
    vec3 ret;
    ret.x = powf(x.x, y.x);
    ret.y = powf(x.y, y.y);
    ret.z = powf(x.z, y.z);
    return ret;
}
inline vec4 LTE_CL_OVERLOADABLE __cl_pow(vec4 x, vec4 y){
    vec4 ret;
    ret.x = powf(x.x, y.x);
    ret.y = powf(x.y, y.y);
    ret.z = powf(x.z, y.z);
    ret.w = powf(x.w, y.w);
    return ret;
}

inline float  LTE_CL_OVERLOADABLE __cl_fract(float x) {
  float tmp;
  float fr = modff(x, &tmp);
  return fr;
}

inline double LTE_CL_OVERLOADABLE __cl_fract(double x) {
  double tmp;
  double fr = modfd(x, &tmp);
  return fr;
}

inline vec2  LTE_CL_OVERLOADABLE __cl_fract(vec2 x) {
  vec2 ret;
  float tmp0, tmp1;
  ret.x = modff(x.x, &tmp0);
  ret.y = modff(x.y, &tmp1);
  return ret;
}

inline vec3  LTE_CL_OVERLOADABLE __cl_fract(vec3 x) {
  float tmp0, tmp1, tmp2;
  vec3 ret;
  ret.x = modff(x.x, &tmp0);
  ret.y = modff(x.y, &tmp1);
  ret.z = modff(x.z, &tmp2);
  return ret;
}

inline vec4  LTE_CL_OVERLOADABLE __cl_fract(vec4 x) {
  float tmp0, tmp1, tmp2, tmp3;
  vec4 ret;
  ret.x = modff(x.x, &tmp0);
  ret.y = modff(x.y, &tmp1);
  ret.z = modff(x.z, &tmp2);
  ret.w = modff(x.w, &tmp3);
  return ret;
}

inline int LTE_CL_OVERLOADABLE __cl_isfinite(float x) { return isfinitef(x); }
inline int LTE_CL_OVERLOADABLE __cl_isfinite(double x) { return isfinited(x); }

inline int LTE_CL_OVERLOADABLE __cl_isinf(float x) { return isinff(x); }
inline int LTE_CL_OVERLOADABLE __cl_isinf(double x) { return isinfd(x); }

inline int LTE_CL_OVERLOADABLE __cl_isnan(float x) { return (x == x) ? 1 : 0; }
inline int LTE_CL_OVERLOADABLE __cl_isnan(double x) { return (x == x) ? 1 : 0; }


inline float  LTE_CL_OVERLOADABLE __cl_sign(float x) {
  //Returns 1.0 if x > 0, -0.0 if x = -0.0, +0.0 if x =
  //+0.0, or –1.0 if x < 0. Returns 0.0 if x is a NaN.
  if (isnan(x)) return (0.0f / 0.0f); // NaN

  if (x > 0.0f) {
    return 1.0f;
  } else if (x == -0.0f) {
    return -0.0f;
  } else if (x == +0.0f) {
    return +0.0f;
  } else if (x < 0.0f) {
    return -1.0f;
  } else {
    // ??? Never happens.
  }

  return (0.0f / 0.0f);
}

inline double  LTE_CL_OVERLOADABLE __cl_sign(double x) {
  //Returns 1.0 if x > 0, -0.0 if x = -0.0, +0.0 if x =
  //+0.0, or –1.0 if x < 0. Returns 0.0 if x is a NaN.
  if (isnan(x)) return (0.0 / 0.0); // NaN

  if (x > 0.0) {
    return 1.0;
  } else if (x == -0.0) {
    return -0.0;
  } else if (x == +0.0) {
    return +0.0;
  } else if (x < 0.0) {
    return -1.0;
  } else {
    // ??? Never happens.
  }
  return (0.0 / 0.0);
}

inline vec2  LTE_CL_OVERLOADABLE __cl_sign(vec2 x) {
  //Returns 1.0 if x > 0, -0.0 if x = -0.0, +0.0 if x =
  //+0.0, or –1.0 if x < 0. Returns 0.0 if x is a NaN.
  vec2 ret;
  if (isnan(x.x)) ret.x = (0.0f / 0.0f); // NaN
  if (isnan(x.y)) ret.y = (0.0f / 0.0f); // NaN

  if (x.x > 0.0f) {
    ret.x = 1.0f;
  } else if (x.x == -0.0f) {
    ret.x = -0.0f;
  } else if (x.x == +0.0f) {
    ret.x = +0.0f;
  } else if (x.x < 0.0f) {
    ret.x = -1.0f;
  } else {
    ret.x = (0.0f / 0.0f);
    // ??? Never happens.
  }

  if (x.y > 0.0f) {
    ret.y = 1.0f;
  } else if (x.y == -0.0f) {
    ret.y = -0.0f;
  } else if (x.y == +0.0f) {
    ret.y = +0.0f;
  } else if (x.y < 0.0f) {
    ret.y = -1.0f;
  } else {
    ret.y = (0.0f / 0.0f);
    // ??? Never happens.
  }

  return ret;
}

inline vec3  LTE_CL_OVERLOADABLE __cl_sign(vec3 x) {
  //Returns 1.0 if x > 0, -0.0 if x = -0.0, +0.0 if x =
  //+0.0, or –1.0 if x < 0. Returns 0.0 if x is a NaN.
  vec3 ret;
  if (isnan(x.x)) ret.x = (0.0f / 0.0f); // NaN
  if (isnan(x.y)) ret.y = (0.0f / 0.0f); // NaN
  if (isnan(x.z)) ret.z = (0.0f / 0.0f); // NaN

  if (x.x > 0.0f) {
    ret.x = 1.0f;
  } else if (x.x == -0.0f) {
    ret.x = -0.0f;
  } else if (x.x == +0.0f) {
    ret.x = +0.0f;
  } else if (x.x < 0.0f) {
    ret.x = -1.0f;
  } else {
    ret.x = (0.0f / 0.0f);
    // ??? Never happens.
  }

  if (x.y > 0.0f) {
    ret.y = 1.0f;
  } else if (x.y == -0.0f) {
    ret.y = -0.0f;
  } else if (x.y == +0.0f) {
    ret.y = +0.0f;
  } else if (x.y < 0.0f) {
    ret.y = -1.0f;
  } else {
    ret.y = (0.0f / 0.0f);
    // ??? Never happens.
  }

  if (x.z > 0.0f) {
    ret.z = 1.0f;
  } else if (x.z == -0.0f) {
    ret.z = -0.0f;
  } else if (x.z == +0.0f) {
    ret.z = +0.0f;
  } else if (x.z < 0.0f) {
    ret.z = -1.0f;
  } else {
    ret.z = (0.0f / 0.0f);
    // ??? Never happens.
  }

  return ret;
}

inline vec4  LTE_CL_OVERLOADABLE __cl_sign(vec4 x) {
  //Returns 1.0 if x > 0, -0.0 if x = -0.0, +0.0 if x =
  //+0.0, or –1.0 if x < 0. Returns 0.0 if x is a NaN.
  vec4 ret;
  if (isnan(x.x)) ret.x = (0.0f / 0.0f); // NaN
  if (isnan(x.y)) ret.y = (0.0f / 0.0f); // NaN
  if (isnan(x.z)) ret.z = (0.0f / 0.0f); // NaN
  if (isnan(x.w)) ret.w = (0.0f / 0.0f); // NaN

  if (x.x > 0.0f) {
    ret.x = 1.0f;
  } else if (x.x == -0.0f) {
    ret.x = -0.0f;
  } else if (x.x == +0.0f) {
    ret.x = +0.0f;
  } else if (x.x < 0.0f) {
    ret.x = -1.0f;
  } else {
    ret.x = (0.0f / 0.0f);
    // ??? Never happens.
  }

  if (x.y > 0.0f) {
    ret.y = 1.0f;
  } else if (x.y == -0.0f) {
    ret.y = -0.0f;
  } else if (x.y == +0.0f) {
    ret.y = +0.0f;
  } else if (x.y < 0.0f) {
    ret.y = -1.0f;
  } else {
    ret.y = (0.0f / 0.0f);
    // ??? Never happens.
  }

  if (x.z > 0.0f) {
    ret.z = 1.0f;
  } else if (x.z == -0.0f) {
    ret.z = -0.0f;
  } else if (x.z == +0.0f) {
    ret.z = +0.0f;
  } else if (x.z < 0.0f) {
    ret.z = -1.0f;
  } else {
    ret.z = (0.0f / 0.0f);
    // ??? Never happens.
  }

  if (x.w > 0.0f) {
    ret.w = 1.0f;
  } else if (x.w == -0.0f) {
    ret.w = -0.0f;
  } else if (x.w == +0.0f) {
    ret.w = +0.0f;
  } else if (x.w < 0.0f) {
    ret.w = -1.0f;
  } else {
    ret.w = (0.0f / 0.0f);
    // ??? Never happens.
  }

  return ret;
}

//
// Shader like functions
//
inline float  LTE_CL_OVERLOADABLE __cl_radians(float degree) { return (M_PI_F / 180.0f) * degree; }
inline double  LTE_CL_OVERLOADABLE __cl_radians(double degree) { return (M_PI / 180.0) * degree; }
inline vec2  LTE_CL_OVERLOADABLE __cl_radians(vec2 degree){
    vec2 ret;
    ret.x = (M_PI / 180.0) * degree.x;
    ret.y = (M_PI / 180.0) * degree.y;
    return ret;
}
inline vec3  LTE_CL_OVERLOADABLE __cl_radians(vec3 degree){
    vec3 ret;
    ret.x = (M_PI / 180.0) * degree.x;
    ret.y = (M_PI / 180.0) * degree.y;
    ret.z = (M_PI / 180.0) * degree.z;
    return ret;
}
inline vec4  LTE_CL_OVERLOADABLE __cl_radians(vec4 degree){
    vec4 ret;
    ret.x = (M_PI / 180.0) * degree.x;
    ret.y = (M_PI / 180.0) * degree.y;
    ret.z = (M_PI / 180.0) * degree.z;
    ret.w = (M_PI / 180.0) * degree.w;
    return ret;
}

    inline float  LTE_CL_OVERLOADABLE __cl_degrees(float radian) { return (180.0f / M_PI_F) * radian; }
inline double  LTE_CL_OVERLOADABLE __cl_degrees(double radian) { return (180.0 / M_PI) * radian; }
inline vec2  LTE_CL_OVERLOADABLE __cl_degrees(vec2 radian){
    vec2 ret;
    ret.x = (180.0 / M_PI) * radian.x;
    ret.y = (180.0 / M_PI) * radian.y;
    return ret;
}
inline vec3  LTE_CL_OVERLOADABLE __cl_degrees(vec3 radian){
    vec3 ret;
    ret.x = (180.0 / M_PI) * radian.x;
    ret.y = (180.0 / M_PI) * radian.y;
    ret.z = (180.0 / M_PI) * radian.z;
    return ret;
}
inline vec4  LTE_CL_OVERLOADABLE __cl_degrees(vec4 radian){
    vec4 ret;
    ret.x = (180.0 / M_PI) * radian.x;
    ret.y = (180.0 / M_PI) * radian.y;
    ret.z = (180.0 / M_PI) * radian.z;
    ret.w = (180.0 / M_PI) * radian.w;
    return ret;
}

inline float  LTE_CL_OVERLOADABLE __cl_clamp(float x, float minval, float maxval) {
  // @todo { use fmin/fmax}
  return min(max(x, minval), maxval);
}

inline double  LTE_CL_OVERLOADABLE __cl_clamp(double x, double minval, double maxval) {
  // @todo { use fmin/fmax}
  return min(max(x, minval), maxval);
}

inline vec2  LTE_CL_OVERLOADABLE __cl_clamp(vec2 x, float minval, float maxval) {
  vec2 ret;
  ret.x = min(max(x.x, minval), maxval);
  ret.y = min(max(x.y, minval), maxval);
  return ret;
}

inline vec2  LTE_CL_OVERLOADABLE __cl_clamp(vec2 x, vec2 minval, vec2 maxval) {
  vec2 ret;
  ret.x = min(max(x.x, minval.x), maxval.x);
  ret.y = min(max(x.y, minval.y), maxval.y);
  return ret;
}

inline vec3  LTE_CL_OVERLOADABLE __cl_clamp(vec3 x, float minval, float maxval) {
  vec3 ret;
  ret.x = min(max(x.x, minval), maxval);
  ret.y = min(max(x.y, minval), maxval);
  ret.z = min(max(x.z, minval), maxval);
  return ret;
}

inline vec3  LTE_CL_OVERLOADABLE __cl_clamp(vec3 x, vec3 minval, vec3 maxval) {
  vec3 ret;
  ret.x = min(max(x.x, minval.x), maxval.x);
  ret.y = min(max(x.y, minval.y), maxval.y);
  ret.z = min(max(x.z, minval.z), maxval.z);
  return ret;
}

inline vec4  LTE_CL_OVERLOADABLE __cl_clamp(vec4 x, float minval, float maxval) {
  vec4 ret;
  ret.x = min(max(x.x, minval), maxval);
  ret.y = min(max(x.y, minval), maxval);
  ret.z = min(max(x.z, minval), maxval);
  ret.w = min(max(x.w, minval), maxval);
  return ret;
}

inline vec4  LTE_CL_OVERLOADABLE __cl_clamp(vec4 x, vec4 minval, vec4 maxval) {
  vec4 ret;
  ret.x = min(max(x.x, minval.x), maxval.x);
  ret.y = min(max(x.y, minval.y), maxval.y);
  ret.z = min(max(x.z, minval.z), maxval.z);
  ret.w = min(max(x.w, minval.w), maxval.w);
  return ret;
}

inline float  LTE_CL_OVERLOADABLE __cl_mix(float x, float y, float t) { return x + (y - x) * t; }
inline double LTE_CL_OVERLOADABLE __cl_mix(double x, double y, double t) { return x + (y - x) * t; }
inline vec2 LTE_CL_OVERLOADABLE __cl_mix(vec2 x, vec2 y, float t) { return x + (y - x) * t; }
inline vec4 LTE_CL_OVERLOADABLE __cl_mix(vec4 x, vec4 y, vec4 t) { return x + (y - x) * t; }
inline vec4 LTE_CL_OVERLOADABLE __cl_mix(vec4 x, vec4 y, float t) { return x + (y - x) * t; }

inline float  LTE_CL_OVERLOADABLE __cl_step(float edge, float x) { return (x < edge) ? 0.0f : 1.0f;; }
inline double  LTE_CL_OVERLOADABLE __cl_step(double edge, double x) { return (x < edge) ? 0.0 : 1.0;; }
inline vec3  LTE_CL_OVERLOADABLE __cl_step(vec3 edge, vec3 x) {
  vec3 ret;
  ret.x = (x.x < edge.x) ? 0.0f : 1.0f;
  ret.y = (x.y < edge.y) ? 0.0f : 1.0f;
  ret.z = (x.z < edge.z) ? 0.0f : 1.0f;
  return ret;
}
inline vec4  LTE_CL_OVERLOADABLE __cl_step(vec4 edge, vec4 x) {
  vec4 ret;
  ret.x = (x.x < edge.x) ? 0.0f : 1.0f;
  ret.y = (x.y < edge.y) ? 0.0f : 1.0f;
  ret.z = (x.z < edge.z) ? 0.0f : 1.0f;
  ret.w = (x.w < edge.w) ? 0.0f : 1.0f;
  return ret;
}

inline float  LTE_CL_OVERLOADABLE __cl_smoothstep(float edge0, float edge1, float x) {
  if (x <= edge0) {
    return 0.0f;
  } else if (x >= edge1) {
    return 1.0f;
  } else {
    float t;
    t = clamp((x - edge0) / (edge1 - edge0), 0.0f, 1.0f);
    return t * t * (3.0f - 2.0f * t);
  }
}

inline double  LTE_CL_OVERLOADABLE __cl_smoothstep(double edge0, double edge1, double x) {
  if (x <= edge0) {
    return 0.0;
  } else if (x >= edge1) {
    return 1.0;
  } else {
    float t;
    t = clamp((x - edge0) / (edge1 - edge0), 0.0, 1.0);
    return t * t * (3.0 - 2.0 * t);
  }
}

inline vec2  LTE_CL_OVERLOADABLE __cl_smoothstep(vec2 edge0, vec2 edge1, vec2 x) {
  vec2 ret;
  if (x.x <= edge0.x) {
    ret.x = 0.0f;
  } else if (x.x >= edge1.x) {
    ret.x = 1.0f;
  } else {
    float t;
    t = clamp((x.x - edge0.x) / (edge1.x - edge0.x), 0.0f, 1.0f);
    ret.x = t * t * (3.0f - 2.0f * t);
  }
  if (x.y <= edge0.y) {
    ret.y = 0.0f;
  } else if (x.y >= edge1.y) {
    ret.y = 1.0f;
  } else {
    float t;
    t = clamp((x.y - edge0.y) / (edge1.y - edge0.y), 0.0f, 1.0f);
    ret.y = t * t * (3.0f - 2.0f * t);
  }
  return ret;
}

inline vec3  LTE_CL_OVERLOADABLE __cl_smoothstep(vec3 edge0, vec3 edge1, vec3 x) {
  vec3 ret;
  if (x.x <= edge0.x) {
    ret.x = 0.0f;
  } else if (x.x >= edge1.x) {
    ret.x = 1.0f;
  } else {
    float t;
    t = clamp((x.x - edge0.x) / (edge1.x - edge0.x), 0.0f, 1.0f);
    ret.x = t * t * (3.0f - 2.0f * t);
  }
  if (x.y <= edge0.y) {
    ret.y = 0.0f;
  } else if (x.y >= edge1.y) {
    ret.y = 1.0f;
  } else {
    float t;
    t = clamp((x.y - edge0.y) / (edge1.y - edge0.y), 0.0f, 1.0f);
    ret.y = t * t * (3.0f - 2.0f * t);
  }
  if (x.z <= edge0.z) {
    ret.z = 0.0f;
  } else if (x.z >= edge1.z) {
    ret.z = 1.0f;
  } else {
    float t;
    t = clamp((x.z - edge0.z) / (edge1.z - edge0.z), 0.0f, 1.0f);
    ret.z = t * t * (3.0f - 2.0f * t);
  }
  return ret;
}

inline vec4  LTE_CL_OVERLOADABLE __cl_smoothstep(vec4 edge0, vec4 edge1, vec4 x) {
  vec4 ret;
  if (x.x <= edge0.x) {
    ret.x = 0.0f;
  } else if (x.x >= edge1.x) {
    ret.x = 1.0f;
  } else {
    float t;
    t = clamp((x.x - edge0.x) / (edge1.x - edge0.x), 0.0f, 1.0f);
    ret.x = t * t * (3.0f - 2.0f * t);
  }
  if (x.y <= edge0.y) {
    ret.y = 0.0f;
  } else if (x.y >= edge1.y) {
    ret.y = 1.0f;
  } else {
    float t;
    t = clamp((x.y - edge0.y) / (edge1.y - edge0.y), 0.0f, 1.0f);
    ret.y = t * t * (3.0f - 2.0f * t);
  }
  if (x.z <= edge0.z) {
    ret.z = 0.0f;
  } else if (x.z >= edge1.z) {
    ret.z = 1.0f;
  } else {
    float t;
    t = clamp((x.z - edge0.z) / (edge1.z - edge0.z), 0.0f, 1.0f);
    ret.z = t * t * (3.0f - 2.0f * t);
  }
  if (x.w <= edge0.w) {
    ret.w = 0.0f;
  } else if (x.w >= edge1.w) {
    ret.w = 1.0f;
  } else {
    float t;
    t = clamp((x.w - edge0.w) / (edge1.w - edge0.w), 0.0f, 1.0f);
    ret.w = t * t * (3.0f - 2.0f * t);
  }
  return ret;
}

//
// Geometric functions
//
inline float  LTE_CL_OVERLOADABLE __cl_dot(float x, float y) { return x * y; }
inline float  LTE_CL_OVERLOADABLE __cl_dot(double x, double y) { return x * y; }
inline float  LTE_CL_OVERLOADABLE __cl_dot(vec2 x, vec2 y) { return x.x * y.x + x.y * y.y; }
inline float  LTE_CL_OVERLOADABLE __cl_dot(vec3 x, vec3 y) { return x.x * y.x + x.y * y.y +x.z * y.z; }
inline float  LTE_CL_OVERLOADABLE __cl_dot(vec4 x, vec4 y) { return x.x * y.x + x.y * y.y +x.z * y.z + x.w * y.w; }

inline vec3  LTE_CL_OVERLOADABLE __cl_cross(vec3 x, vec3 y) {
  vec3 r;
  r.x = x[2] * y[1] - x[1] * y[2];
  r.y = x[0] * y[2] - x[2] * y[0];
  r.z = x[1] * y[0] - x[0] * y[1];
  return r;
}

inline vec4  LTE_CL_OVERLOADABLE __cl_cross(vec4 x, vec4 y) {
  vec4 r;
  r.x = x[2] * y[1] - x[1] * y[2];
  r.y = x[0] * y[2] - x[2] * y[0];
  r.z = x[1] * y[0] - x[0] * y[1];
  r.w = 0.0f;
  return r;
}

inline float LTE_CL_OVERLOADABLE __cl_length(float x) {
  return sqrt(__cl_dot(x, x));
}
inline float LTE_CL_OVERLOADABLE __cl_length(double x) {
  return sqrt(__cl_dot(x, x));
}
inline float LTE_CL_OVERLOADABLE __cl_length(vec2 x) {
  return sqrt(__cl_dot(x, x));
}
inline float LTE_CL_OVERLOADABLE __cl_length(vec3 x) {
  return sqrt(__cl_dot(x, x));
}
inline float LTE_CL_OVERLOADABLE __cl_length(vec4 x) {
  return sqrt(__cl_dot(x, x));
}

inline float LTE_CL_OVERLOADABLE __cl_distance(float x, float y) {
  return sqrt(__cl_dot(x-y, x-y));
}
inline float LTE_CL_OVERLOADABLE __cl_distance(double x, double y) {
  return sqrt(__cl_dot(x-y, x-y));
}
inline float LTE_CL_OVERLOADABLE __cl_distance(vec2 x, vec2 y) {
  return sqrt(__cl_dot(x-y, x-y));
}
inline float LTE_CL_OVERLOADABLE __cl_distance(vec3 x, vec3 y) {
  return sqrt(__cl_dot(x-y, x-y));
}
inline float LTE_CL_OVERLOADABLE __cl_distance(vec4 x, vec4 y) {
  return sqrt(__cl_dot(x-y, x-y));
}


inline float LTE_CL_OVERLOADABLE __cl_normalize(float x) {
  // Igonore .w component.
  float len = sqrt((x * x));
  float r = x;
  if (len > 1.0e-30) {
    float inv_len = 1.0 / len;
    r *= inv_len;
  }
  return r;
}

inline vec2 LTE_CL_OVERLOADABLE __cl_normalize(vec2 x) {
  // Igonore .w component.
  float len = sqrt((x.x * x.x + x.y * x.y));
  vec2 r = x;
  if (len > 1.0e-30) {
    float inv_len = 1.0 / len;
    r.x *= inv_len;
    r.y *= inv_len;
  }
  return r;
}

inline vec3 LTE_CL_OVERLOADABLE __cl_normalize(vec3 x) {
  // Igonore .w component.
  float len = sqrt((x.x * x.x + x.y * x.y + x.z * x.z));
  vec3 r = x;
  if (len > 1.0e-30) {
    float inv_len = 1.0 / len;
    r.x *= inv_len;
    r.y *= inv_len;
    r.z *= inv_len;
  }
  return r;
}

inline vec4 LTE_CL_OVERLOADABLE __cl_normalize(vec4 x) {
  // Igonore .w component.
  float len = sqrt((x.x * x.x + x.y * x.y + x.z * x.z));
  vec4 r = x;
  if (len > 1.0e-30) {
    float inv_len = 1.0 / len;
    r.x *= inv_len;
    r.y *= inv_len;
    r.z *= inv_len;
  }
  return r;
}

//#define abs_diff       _cl_abs_diff
//#define acospi         _cl_acospi
//#define add_sat        _cl_add_sat
//#define all            _cl_all
//#define any            _cl_any
//#define asinpi         _cl_asinpi
//#define atan2pi        _cl_atan2pi
//#define atanpi         _cl_atanpi
//#define bitselect      _cl_bitselect
//#define cbrt           _cl_cbrt
//#define clz            _cl_clz
//#define copysign       _cl_copysign
//#define cospi          _cl_cospi
//#define erf            _cl_erf
//#define erfc           _cl_erfc
//#define exp10          _cl_exp10
//#define expm1          _cl_expm1
//#define fdim           _cl_fdim
//#define fmax           _cl_fmax
//#define fmin           _cl_fmin
//#define frexp          _cl_frexp
//#define hadd           _cl_hadd
//#define hypot          _cl_hypot
//#define ilogb          _cl_ilogb
//#define isequal        _cl_isequal
//#define isgreater      _cl_isgreater
//#define isgreaterequal _cl_isgreaterequal
//#define isless         _cl_isless
//#define islessequal    _cl_islessequal
//#define islessgreater  _cl_islessgreater
//#define isnormal       _cl_isnormal
//#define isnotequal     _cl_isnotequal
//#define isordered      _cl_isordered
//#define isunordered    _cl_isunordered
//#define ldexp          _cl_ldexp
//#define ldexp          _cl_ldexp
//#define lgamma         _cl_lgamma
//#define lgamma_r       _cl_lgamma_r
//#define log10          _cl_log10
//#define log1p          _cl_log1p
//#define logb           _cl_logb
//#define mad            _cl_mad
//#define mad24          _cl_mad24
//#define mad_hi         _cl_mad_hi
//#define mad_sat        _cl_mad_sat
//#define maxmag         _cl_maxmag
//#define minmag         _cl_minmag
//#define modf           _cl_modf
//#define mul24          _cl_mul24
//#define mul_hi         _cl_mul_hi
//#define nan            _cl_nan
//#define nextafter      _cl_nextafter
//#define popcount       _cl_popcount
//#define pown           _cl_pown
//#define pown           _cl_pown
//#define powr           _cl_powr
//#define remainder      _cl_remainder
//#define remquo         _cl_remquo
//#define rhadd          _cl_rhadd
//#define rint           _cl_rint
//#define rootn          _cl_rootn
//#define rootn          _cl_rootn
//#define rotate         _cl_rotate
//#define round          _cl_round
//#define rsqrt          _cl_rsqrt
//#define select         _cl_select
//#define signbit        _cl_signbit
//#define sincos         _cl_sincos
//#define sinpi          _cl_sinpi
//#define sub_sat        _cl_sub_sat
//#define tanpi          _cl_tanpi
//#define tgamma         _cl_tgamma
//#define trunc          _cl_trunc
//#define upsample       _cl_upsample

// IO function
int   printf(const char *fmt, ...);

#else
#define LTE_EXTERN  extern

LTE_EXTERN double fmad(double, double, double);
LTE_EXTERN double fmodd(double, double);
LTE_EXTERN double sqrtd(double);
LTE_EXTERN double fabsd(double);
LTE_EXTERN double acosd(double);
LTE_EXTERN double acoshd(double);
LTE_EXTERN double asind(double);
LTE_EXTERN double asinhd(double);
LTE_EXTERN double atand(double);
LTE_EXTERN double atan2d(double, double);
LTE_EXTERN double atanhd(double);
LTE_EXTERN double sind(double);
LTE_EXTERN double cosd(double);
LTE_EXTERN double tand(double);
LTE_EXTERN double expd(double);
LTE_EXTERN double exp2d(double);
LTE_EXTERN double logd(double);
LTE_EXTERN double log2d(double);
LTE_EXTERN double floord(double);
LTE_EXTERN double ceild(double);
LTE_EXTERN double powd(double, double);

LTE_EXTERN double modfd(double, double*);

LTE_EXTERN int isinfd(double);
LTE_EXTERN int isfinited(double);

#endif

LTE_EXTERN void myfunc(int a);
LTE_EXTERN float trace(ShaderEnv* e, const vec4* org, const vec4* dir, vec4* cs);
LTE_EXTERN int texture2D(ShaderEnv* e, int texID, float u, float v, vec4* cs);
LTE_EXTERN int texture2DDegamma(ShaderEnv* e, int texID, float u, float v, float degamma, vec4* cs);
LTE_EXTERN int textureGrad2D(ShaderEnv* e, int texID, float u, float v, vec4* cs00, vec4* cs10, vec4* cs01);
LTE_EXTERN void envmap3D(ShaderEnv* e, const vec4* dir, vec4* cs);
LTE_EXTERN void computeFresnel(ShaderEnv* e, vec4* refl, vec4* refr, float* kr, float* kt, const vec4* wi, const vec4* n, float eta, vec4* cs);
LTE_EXTERN void transmittanceSSS(ShaderEnv* e, const vec4* P, const vec4* N, const vec4* lightP, float translucency, float sssWidth, float dist, vec4* cs);
LTE_EXTERN void sunsky(ShaderEnv* e, float theta, float phi, vec4* cs);
LTE_EXTERN void fetchDiffuse(ShaderEnv* e, float u, float v, float falloff, vec4* cs);
LTE_EXTERN void fetchReflection(ShaderEnv* e, float u, float v, float falloff, vec4* cs);
LTE_EXTERN void fetchRefraction(ShaderEnv* e, float u, float v, float falloff, vec4* cs);

//
// Light
//
LTE_EXTERN int  getNumLights();
LTE_EXTERN const Light* getLight(int i);
LTE_EXTERN int  sampleLight(vec4* pos, vec4* dir);

//
// VPL
//
LTE_EXTERN int  getNumVPLs();
LTE_EXTERN Light* getVPL(int i);


//
// Param
//
//extern int getParamInt(void* param, const char* name, int *value);
LTE_EXTERN int getParamFloat(ShaderEnv* env, const char* name, float* value);
LTE_EXTERN int getParamFloat3(ShaderEnv* param, const char* name, vec4* value);
//extern int getParamString(void* param, const char* name, char* value);

//
// System
//
LTE_EXTERN int getNumThreads();

//
// Random
//
LTE_EXTERN float randomreal();

//
// Photonmap
//

// Returns diffuse value and # of photons found.
LTE_EXTERN int photondiffuse(const vec4* p, const vec4* n, vec4* cs);

//
// For Imager
// 
LTE_EXTERN int getImageSize(int* width, int* height);


#ifdef __cplusplus
}
#endif

#endif
