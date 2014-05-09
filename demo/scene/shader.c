//
// Online shader editor and interactive raytracing preview.
// 
// You can use OpenCL C like language and some built-in functions. See:
//   http://www.khronos.org/registry/cl/sdk/1.2/docs/man/xhtml/
//
// Rev: Jan 17, 2014
//
// LTE Inc.
//

#include "shader.h"
#include "procedural-noise.c"

inline vec4 vmax(vec4 v, float maxval)
{
    vec4 r = v;
    if (r[0] > maxval) r[0] = maxval;
    if (r[1] > maxval) r[1] = maxval;
    if (r[2] > maxval) r[2] = maxval;

    return r;
}

inline float vdot3(vec4 a, vec4 b)
{
    return a[0] * b[0] + a[1] * b[1] + a[2] * b[2];
}

inline void sample_arealight(vec4* ret, vec4 p, float radius)
{
    ret->x = p.x + radius * randomreal();
    ret->y = p.y + radius * randomreal();
    ret->z = p.z;

}


void shader(ShaderEnv *env)
{
  // Simple area light
  vec4 lightvec;

  vec4 lightP;
  lightP[0] = 10.0;
  lightP[1] = 30.0;
  lightP[2] = 10.0;

  vec4 arealightP;
  float arealightSize = 15.0;
  sample_arealight(&arealightP, lightP, arealightSize);
  lightvec[0] = arealightP[0];
  lightvec[1] = arealightP[1];
  lightvec[2] = arealightP[2];

  lightvec = normalize(lightvec);

  // diffuse s
  vec4 n = env->N;
  // faceforward
  float idotn = vdot3(env->Ng, env->I);
  if (idotn > 0.0) {
    n = n;
  }

  float vdotl = vdot3(lightvec, n);
  vdotl = clamp(vdotl, 0.3f, 1.0f);


  // terminate ray if depth > 1
  if (env->trace_depth > 1) {
      float coeff = fabs(vdot3(normalize(env->I), -n));
      env->Ci = coeff * env->diffuse;   
    return;
  }
  //
  // trace shadow ray
  //
  vec4 dir = lightvec;

  vec4 org;
  org = env->P + 0.1f * n;
  
  vec4 col;
  float dist = trace(env, &org, &dir, &col);

  if (dist > 0.0) {
    // shadow 
    env->Ci = 0.2 * vdotl;
  } else {
    // no shadow
    env->Ci = vdotl;
  }

  float aCoeff = 0.0f; 
  int aSampleMax = 32;
  float aDistMax = 80.0f;
  
  for( int i = 0; i < aSampleMax; ++i){
    float phi = 3.141592f * 2.0f * randomreal();
    float theta = randomreal();
    float thetasq = sqrtf( theta );
    

  //vec4 tv = env->tangent;
  //vec4 bv = env->binormal;

    int la = 0; la = n[0] < n[1] ? 1: 0; la = la < n[2] ? 2 : la;
    vec4 tv; 
    tv[ ( la + 1 ) % 3] = 1.0f;
    vec4 bv = cross( n, tv );
    bv = normalize( bv );
    tv = cross( bv, n );
    
    dir = tv * ( cos( phi ) * thetasq );
    dir = dir + bv * ( sin(phi) * thetasq );
    dir = dir + n * sqrtf( 1.0f - theta );

    dist = trace( env, &org, &dir, 0 );
    if( 0.0f <= dist && dist <= aDistMax){
      aCoeff += dist / aDistMax;
    }else{
      aCoeff += 1.0f;
    }
  }
  aCoeff /= (float)aSampleMax;
  env->Ci *= aCoeff;
  
  // envmap
  vec4 rdir, tdir;
  float kr, kt;

  float eta = 1.4;
  // fresnel
  vec4 indir = normalize(env->I);
  computeFresnel(env, &rdir, &tdir, &kr, &kt, &indir, &n, eta, 0);

  // env shadow check.
  org = env->P + 0.1f * rdir;
  dir = rdir;
  dist = trace(env, &org, &dir, 0);
  float env_power = 1.0;
  if (dist > 0.0) {
    // hit geom.
    env_power = 0.0f;
  } else {
    // hit sky.
    env_power = 1.0f;
  }

  float frequency = 3;
  
  vec4 blackcolor;
  blackcolor[0]=1.0;
  blackcolor[1]=0.2;
  blackcolor[2]=0.2;

  float smod = fmod (env->s* frequency, 1);
  float tmod = fmod (env->t* frequency, 1);
	
	if (smod < 0.5) {
		if (tmod < 0.5)
			env->Ci = env->diffuse;
		else
			env->Ci = blackcolor;
	}
	else {
		if (tmod < 0.5)
			env->Ci = blackcolor;
		else
			env->Ci = env->diffuse;
	}
	
	float inTop;
  vec4 redcolor;
  redcolor[0]=1.0;
  redcolor[1]=1.0;
  redcolor[2]=0.0;

  vec4 greencolor;
  greencolor[0]=0.0;
  greencolor[1]=1.0;
  greencolor[2]=0.0;
  
  inTop = smoothstep(0.3f, 0.7f, env->t);
  
  vec4 other_col;
  other_col = mix(greencolor,redcolor,inTop);
  
	//if (env->t < 0.5)
  //   other_col=greencolor;
  //else
  //   other_col=redcolor;
  
  
  vec4 env_col;
  env_col[0] = 0.5;
  env_col[1] = 0.5;
  env_col[2] = 1.0;
  env_col[3] = 0.0;
  // = vec4(1, 0, 0, 0);
  rdir = -rdir;
  //envmap3D(env, &rdir, &env_col);
  //env_col = vmax(env_col, 10.0f);
  //while (env_col[1]) {
  //  env->Ci = vdotl * env_col * aCoeff;
  //}
  
  //loop(env_col);
  // output
  //env->Ci = env->Ci * env->diffuse + env->ambient +  kr * env_col;
  //env->Ci = env_col;
  //env->Ci = other_col+ env->ambient + env_power * kr * env_col;
  //env->Ci = vdotl * env_col;
  env->Ci = vdotl * env_col * aCoeff ;
  //env->Ci = aCoeff;
  
  //vec3 p = snoise3d(env->P.xyz);
  //env->Ci.xyz += p;
}