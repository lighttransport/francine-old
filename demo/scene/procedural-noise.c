//
// GLSL textureless classic 3D noise "cnoise",
// with an RSL-style periodic variant "pnoise".
// Author:  Stefan Gustavson (stefan.gustavson@liu.se)
// Version: 2011-10-11
//
// Many thanks to Ian McEwan of Ashima Arts for the
// ideas for permutation and gradient selection.
//
// Copyright (c) 2011 Stefan Gustavson. All rights reserved.
// Distributed under the MIT license. See LICENSE file.
// https://github.com/ashima/webgl-noise

//
// Description : Array and textureless GLSL 2D/3D/4D simplex 
//               noise functions.
//      Author : Ian McEwan, Ashima Arts.
//  Maintainer : ijm
//     Lastmod : 20110822 (ijm)
//     License : Copyright (C) 2011 Ashima Arts. All rights reserved.
//               Distributed under the MIT License. See LICENSE file.
//               https://github.com/ashima/webgl-noise
// 

// Modified by LTE Inc. to fit light transport shader language syntax.

float mod289_1(float x) {
  return x - floor(x * (1.0f / 289.0f)) * 289.0f;
}

vec2 mod289_2(vec2 x) {
  return x - floor(x * (1.0f / 289.0f)) * 289.0f;
}

vec3 mod289_3(vec3 x) {
  return x - floor(x * (1.0f / 289.0f)) * 289.0f;
}

vec4 mod289_4(vec4 x) {
  return x - floor(x * (1.0f / 289.0f)) * 289.0f;
}

float permutef(float x) {
  return mod289_1(((x*34.0)+1.0)*x);
}

vec3 permute3(vec3 x) {
  return mod289_3(((x*34.0f)+1.0f)*x);
}

vec4 permute4(vec4 x) {
  return mod289_4(((x*34.0f)+1.0f)*x);
}

vec4 taylorInvSqrt(vec4 r)
{
  return 1.79284291400159f - 0.85373472095314f * r;
}

float taylorInvSqrtf(float r)
{
  return 1.79284291400159f - 0.85373472095314f * r;
}

vec3 fade(vec3 t) {
    return t*t*t*(t*(t*6.0f-15.0f)+10.0f);
}

vec4 grad4(float j, vec4 ip)
{
  const vec4 ones = (vec4){1.0f, 1.0f, 1.0f, -1.0f};
  vec4 p,s;

  p.xyz = floor( fract ((vec3){j, j, j} * ip.xyz) * 7.0f) * ip.z - 1.0f;
  p.w = 1.5f - dot(fabs(p.xyz), ones.xyz);
  //s = vec4(lessThan(p, vec4(0.0)));
  s.x = (p.x < 0.0f) ? 1.0f : 0.0f;
  s.y = (p.y < 0.0f) ? 1.0f : 0.0f;
  s.z = (p.z < 0.0f) ? 1.0f : 0.0f;
  s.w = (p.w < 0.0f) ? 1.0f : 0.0f;
  p.xyz = p.xyz + (s.xyz*2.0f - 1.0f) * s.www; 

  return p;
}
            
// Classic Perlin noise
float cnoise(vec3 P)
{
  vec3 Pi0 = floor(P); // Integer part for indexing
  vec3 Pi1 = Pi0 + (vec3){1.0f, 1.0f, 1.0f}; // Integer part + 1
  Pi0 = mod289_3(Pi0);
  Pi1 = mod289_3(Pi1);
  vec3 Pf0 = fract(P); // Fractional part for interpolation
  vec3 Pf1 = Pf0 - (vec3){1.0f, 1.0f, 1.0f}; // Fractional part - 1.0
  vec4 ix = (vec4){Pi0.x, Pi1.x, Pi0.x, Pi1.x};
  vec4 iy = (vec4){Pi0.y, Pi0.y, Pi1.y, Pi1.y};
  vec4 iz0 = Pi0.zzzz;
  vec4 iz1 = Pi1.zzzz;

  vec4 ixy = permute4(permute4(ix) + iy);
  vec4 ixy0 = permute4(ixy + iz0);
  vec4 ixy1 = permute4(ixy + iz1);

  vec4 gx0 = ixy0 * (1.0f / 7.0f);
  vec4 gy0 = fract(floor(gx0) * (1.0f / 7.0f)) - 0.5f;
  gx0 = fract(gx0);
  vec4 gz0 = (vec4){0.5f, 0.5f, 0.5f, 0.5f} - fabs(gx0) - fabs(gy0);
  vec4 sz0 = step(gz0, (vec4){0.0f, 0.0f, 0.0f, 0.0f});
  gx0 -= sz0 * (step(0.0f, gx0) - 0.5f);
  gy0 -= sz0 * (step(0.0f, gy0) - 0.5f);

  vec4 gx1 = ixy1 * (1.0f / 7.0f);
  vec4 gy1 = fract(floor(gx1) * (1.0f / 7.0f)) - 0.5f;
  gx1 = fract(gx1);
  vec4 gz1 = (vec4){0.5f, 0.5f, 0.5f, 0.5f} - fabs(gx1) - fabs(gy1);
  vec4 sz1 = step(gz1, (vec4){0.0f, 0.0f, 0.0f, 0.0f});
  gx1 -= sz1 * (step(0.0f, gx1) - 0.5f);
  gy1 -= sz1 * (step(0.0f, gy1) - 0.5f);

  vec3 g000 = (vec3){gx0.x,gy0.x,gz0.x};
  vec3 g100 = (vec3){gx0.y,gy0.y,gz0.y};
  vec3 g010 = (vec3){gx0.z,gy0.z,gz0.z};
  vec3 g110 = (vec3){gx0.w,gy0.w,gz0.w};
  vec3 g001 = (vec3){gx1.x,gy1.x,gz1.x};
  vec3 g101 = (vec3){gx1.y,gy1.y,gz1.y};
  vec3 g011 = (vec3){gx1.z,gy1.z,gz1.z};
  vec3 g111 = (vec3){gx1.w,gy1.w,gz1.w};

  vec4 norm0 = taylorInvSqrt((vec4){dot(g000, g000), dot(g010, g010), dot(g100, g100), dot(g110, g110)});
  g000 *= norm0.x;
  g010 *= norm0.y;
  g100 *= norm0.z;
  g110 *= norm0.w;
  vec4 norm1 = taylorInvSqrt((vec4){dot(g001, g001), dot(g011, g011), dot(g101, g101), dot(g111, g111)});
  g001 *= norm1.x;
  g011 *= norm1.y;
  g101 *= norm1.z;
  g111 *= norm1.w;

  float n000 = dot(g000, Pf0);
  float n100 = dot(g100, (vec3){Pf1.x, Pf0.y, Pf0.z});
  float n010 = dot(g010, (vec3){Pf0.x, Pf1.y, Pf0.z});
  float n110 = dot(g110, (vec3){Pf1.x, Pf1.y, Pf0.z});
  float n001 = dot(g001, (vec3){Pf0.x, Pf0.y, Pf1.z});
  float n101 = dot(g101, (vec3){Pf1.x, Pf0.y, Pf1.z});
  float n011 = dot(g011, (vec3){Pf0.x, Pf1.y, Pf1.z});
  float n111 = dot(g111, Pf1);

  vec3 fade_xyz = fade(Pf0);
  vec4 n_z = mix((vec4){n000, n100, n010, n110}, (vec4){n001, n101, n011, n111}, fade_xyz.z);
  vec2 n_yz = mix(n_z.xy, n_z.zw, fade_xyz.y);
  float n_xyz = mix(n_yz.x, n_yz.y, fade_xyz.x); 
  return 2.2f * n_xyz;
}

// Classic Perlin noise, periodic variant
float pnoise(vec3 P, vec3 rep)
{
  vec3 Pi0 = fmod(floor(P), rep); // Integer part, modulo period
  vec3 Pi1 = fmod(Pi0 + (vec3){1.0f, 1.0f, 1.0f}, rep); // Integer part + 1, mod period
  Pi0 = mod289_3(Pi0);
  Pi1 = mod289_3(Pi1);
  vec3 Pf0 = fract(P); // Fractional part for interpolation
  vec3 Pf1 = Pf0 - (vec3){1.0f, 1.0f, 1.0f}; // Fractional part - 1.0
  vec4 ix = (vec4){Pi0.x, Pi1.x, Pi0.x, Pi1.x};
  vec4 iy = (vec4){Pi0.y, Pi0.y, Pi1.y, Pi1.y};
  vec4 iz0 = Pi0.zzzz;
  vec4 iz1 = Pi1.zzzz;

  vec4 ixy = permute4(permute4(ix) + iy);
  vec4 ixy0 = permute4(ixy + iz0);
  vec4 ixy1 = permute4(ixy + iz1);

  vec4 gx0 = ixy0 * (1.0f / 7.0f);
  vec4 gy0 = fract(floor(gx0) * (1.0f / 7.0f)) - 0.5f;
  gx0 = fract(gx0);
  vec4 gz0 = (vec4){0.5f, 0.5f, 0.5f, 0.5f} - fabs(gx0) - fabs(gy0);
  vec4 sz0 = step(gz0, (vec4){0.0f, 0.0f, 0.0f, 0.0f});
  gx0 -= sz0 * (step(0.0f, gx0) - 0.5f);
  gy0 -= sz0 * (step(0.0f, gy0) - 0.5f);

  vec4 gx1 = ixy1 * (1.0f / 7.0f);
  vec4 gy1 = fract(floor(gx1) * (1.0f / 7.0f)) - 0.5f;
  gx1 = fract(gx1);
  vec4 gz1 = (vec4){0.5f, 0.5f, 0.5f, 0.5f} - fabs(gx1) - fabs(gy1);
  vec4 sz1 = step(gz1, (vec4){0.0f, 0.0f, 0.0f, 0.0f});
  gx1 -= sz1 * (step(0.0f, gx1) - 0.5f);
  gy1 -= sz1 * (step(0.0f, gy1) - 0.5f);

  vec3 g000 = (vec3){gx0.x,gy0.x,gz0.x};
  vec3 g100 = (vec3){gx0.y,gy0.y,gz0.y};
  vec3 g010 = (vec3){gx0.z,gy0.z,gz0.z};
  vec3 g110 = (vec3){gx0.w,gy0.w,gz0.w};
  vec3 g001 = (vec3){gx1.x,gy1.x,gz1.x};
  vec3 g101 = (vec3){gx1.y,gy1.y,gz1.y};
  vec3 g011 = (vec3){gx1.z,gy1.z,gz1.z};
  vec3 g111 = (vec3){gx1.w,gy1.w,gz1.w};

  vec4 norm0 = taylorInvSqrt((vec4){dot(g000, g000), dot(g010, g010), dot(g100, g100), dot(g110, g110)});
  g000 *= norm0.x;
  g010 *= norm0.y;
  g100 *= norm0.z;
  g110 *= norm0.w;
  vec4 norm1 = taylorInvSqrt((vec4){dot(g001, g001), dot(g011, g011), dot(g101, g101), dot(g111, g111)});
  g001 *= norm1.x;
  g011 *= norm1.y;
  g101 *= norm1.z;
  g111 *= norm1.w;

  float n000 = dot(g000, Pf0);
  float n100 = dot(g100, (vec3){Pf1.x, Pf0.y, Pf0.z});
  float n010 = dot(g010, (vec3){Pf0.x, Pf1.y, Pf0.z});
  float n110 = dot(g110, (vec3){Pf1.x, Pf1.y, Pf0.z});
  float n001 = dot(g001, (vec3){Pf0.x, Pf0.y, Pf1.z});
  float n101 = dot(g101, (vec3){Pf1.x, Pf0.y, Pf1.z});
  float n011 = dot(g011, (vec3){Pf0.x, Pf1.y, Pf1.z});
  float n111 = dot(g111, Pf1);

  vec3 fade_xyz = fade(Pf0);
  vec4 n_z = mix((vec4){n000, n100, n010, n110}, (vec4){n001, n101, n011, n111}, fade_xyz.z);
  vec2 n_yz = mix(n_z.xy, n_z.zw, fade_xyz.y);
  float n_xyz = mix(n_yz.x, n_yz.y, fade_xyz.x); 
  return 2.2f * n_xyz;
}

// Cellnoise 2D
float snoise2d(vec2 v) {
  const vec4 C = (vec4){0.211324865405187f,  // (3.0-sqrt(3.0))/6.0
                        0.366025403784439f,  // 0.5*(sqrt(3.0)-1.0)
                       -0.577350269189626f,  // -1.0 + 2.0 * C.x
                        0.024390243902439f}; // 1.0 / 41.0
// First corner
  vec2 i  = floor(v + dot(v, C.yy) );
  vec2 x0 = v -   i + dot(i, C.xx);

// Other corners
  vec2 i1;
  //i1.x = step( x0.y, x0.x ); // x0.x > x0.y ? 1.0 : 0.0
  //i1.y = 1.0 - i1.x;
  i1 = (x0.x > x0.y) ? (vec2){1.0f, 0.0f} : (vec2){0.0f, 1.0f};
  // x0 = x0 - 0.0 + 0.0 * C.xx ;
  // x1 = x0 - i1 + 1.0 * C.xx ;
  // x2 = x0 - 1.0 + 2.0 * C.xx ;
  vec4 x12 = x0.xyxy + C.xxzz;
  x12.xy -= i1;

// Permutations
  i = mod289_2(i); // Avoid truncation effects in permutation
  vec3 p = permute3( permute3( i.y + (vec3){0.0f, i1.y, 1.0f })
		+ i.x + (vec3){0.0f, i1.x, 1.0f });

  vec3 m = max(0.5f - (vec3){dot(x0,x0), dot(x12.xy,x12.xy), dot(x12.zw,x12.zw)}, 0.0f);
  m = m*m ;
  m = m*m ;

// Gradients: 41 points uniformly over a line, mapped onto a diamond.
// The ring size 17*17 = 289 is close to a multiple of 41 (41*7 = 287)

  vec3 x = 2.0f * fract(p * C.www) - 1.0f;
  vec3 h = fabs(x) - 0.5f;
  vec3 ox = floor(x + 0.5f);
  vec3 a0 = x - ox;

// Normalise gradients implicitly by scaling m
// Approximation of: m *= inversesqrt( a0*a0 + h*h );
  m *= 1.79284291400159f - 0.85373472095314f * ( a0*a0 + h*h );

// Compute final noise value at P
  vec3 g;
  g.x  = a0.x  * x0.x  + h.x  * x0.y;
  g.yz = a0.yz * x12.xz + h.yz * x12.yw;
  return 130.0 * dot(m, g);
}


// Cellnoise
float snoise3d(vec3 v)
  { 
  const vec2  C = (vec2){1.0f/6.0f, 1.0f/3.0f};
  const vec4  D = (vec4){0.0f, 0.5f, 1.0f, 2.0f};

// First corner
  vec3 i  = floor(v + dot(v, C.yyy) );
  vec3 x0 =   v - i + dot(i, C.xxx) ;

// Other corners
  vec3 g = step(x0.yzx, x0.xyz);
  vec3 l = 1.0f - g;
  vec3 i1 = min( g.xyz, l.zxy );
  vec3 i2 = max( g.xyz, l.zxy );

  //   x0 = x0 - 0.0 + 0.0 * C.xxx;
  //   x1 = x0 - i1  + 1.0 * C.xxx;
  //   x2 = x0 - i2  + 2.0 * C.xxx;
  //   x3 = x0 - 1.0 + 3.0 * C.xxx;
  vec3 x1 = x0 - i1 + C.xxx;
  vec3 x2 = x0 - i2 + C.yyy; // 2.0*C.x = 1/3 = C.y
  vec3 x3 = x0 - D.yyy;      // -1.0+3.0*C.x = -0.5 = -D.y

// Permutations
  i = mod289_3(i); 
  vec4 p = permute4( permute4( permute4( 
             i.z + (vec4){0.0f, i1.z, i2.z, 1.0f })
           + i.y + (vec4){0.0f, i1.y, i2.y, 1.0f }) 
           + i.x + (vec4){0.0f, i1.x, i2.x, 1.0f });

// Gradients: 7x7 points over a square, mapped onto an octahedron.
// The ring size 17*17 = 289 is close to a multiple of 49 (49*6 = 294)
  float n_ = 0.142857142857f; // 1.0/7.0
  vec3  ns = n_ * D.wyz - D.xzx;

  vec4 j = p - 49.0f * floor(p * ns.z * ns.z);  //  mod(p,7*7)

  vec4 x_ = floor(j * ns.z);
  vec4 y_ = floor(j - 7.0f * x_ );    // mod(j,N)

  vec4 x = x_ *ns.x + ns.yyyy;
  vec4 y = y_ *ns.x + ns.yyyy;
  vec4 h = 1.0f - fabs(x) - fabs(y);

  vec4 b0 = (vec4){ x.x, x.y, y.x, y.y };
  vec4 b1 = (vec4){ x.z, x.w, y.z, y.w };

  //vec4 s0 = vec4(lessThan(b0,0.0))*2.0 - 1.0;
  //vec4 s1 = vec4(lessThan(b1,0.0))*2.0 - 1.0;
  vec4 s0 = floor(b0)*2.0f + 1.0f;
  vec4 s1 = floor(b1)*2.0f + 1.0f;
  vec4 sh = -step(h, (vec4){0.0f, 0.0f, 0.0f, 0.0f});

  vec4 a0 = b0.xzyw + s0.xzyw*sh.xxyy ;
  vec4 a1 = b1.xzyw + s1.xzyw*sh.zzww ;

  vec3 p0 = (vec3){a0.x, a0.y,h.x};
  vec3 p1 = (vec3){a0.z, a0.w,h.y};
  vec3 p2 = (vec3){a1.x, a1.y,h.z};
  vec3 p3 = (vec3){a1.z, a1.w,h.w};

//Normalise gradients
  vec4 dp = (vec4){dot(p0,p0), dot(p1,p1), dot(p2, p2), dot(p3,p3)};
  vec4 norm = taylorInvSqrt(dp);
  p0 *= norm.x;
  p1 *= norm.y;
  p2 *= norm.z;
  p3 *= norm.w;

// Mix final noise value
  vec4 dx = (vec4){dot(x0,x0), dot(x1,x1), dot(x2,x2), dot(x3,x3)};
  vec4 m = max(0.6f - dx, 0.0f);
  m = m * m;
  vec4 dpx = (vec4){ dot(p0,x0), dot(p1,x1), dot(p2,x2), dot(p3,x3) };

  return 42.0f * dot( m*m, dpx);
}

float snoise4d(vec4 v)
  {
  const vec4  C = (vec4){ 0.138196601125011f,  // (5 - sqrt(5))/20  G4
                          0.276393202250021f,  // 2 * G4
                          0.414589803375032f,  // 3 * G4
                         -0.447213595499958f}; // -1 + 4 * G4

  // (sqrt(5) - 1)/4 = F4, used once below
  const float F4 = 0.309016994374947451f;

// First corner
  vec4 i  = floor(v + dot(v, (vec4){F4, F4, F4, F4}) );
  vec4 x0 = v -   i + dot(i, C.xxxx);

// Other corners

// Rank sorting originally contributed by Bill Licea-Kane, AMD (formerly ATI)
  vec4 i0;
  vec3 isX = step( x0.yzw, x0.xxx );
  vec3 isYZ = step( x0.zww, x0.yyz );
//  i0.x = dot( isX, vec3( 1.0 ) );
  i0.x = isX.x + isX.y + isX.z;
  i0.yzw = 1.0f - isX;
//  i0.y += dot( isYZ.xy, vec2( 1.0 ) );
  i0.y += isYZ.x + isYZ.y;
  i0.zw += 1.0f - isYZ.xy;
  i0.z += isYZ.z;
  i0.w += 1.0f - isYZ.z;

  // i0 now contains the unique values 0,1,2,3 in each channel
  vec4 i3 = clamp( i0, 0.0f, 1.0f );
  vec4 i2 = clamp( i0-1.0f, 0.0f, 1.0f );
  vec4 i1 = clamp( i0-2.0f, 0.0f, 1.0f );

  //  x0 = x0 - 0.0 + 0.0 * C.xxxx
  //  x1 = x0 - i1  + 1.0 * C.xxxx
  //  x2 = x0 - i2  + 2.0 * C.xxxx
  //  x3 = x0 - i3  + 3.0 * C.xxxx
  //  x4 = x0 - 1.0 + 4.0 * C.xxxx
  vec4 x1 = x0 - i1 + C.xxxx;
  vec4 x2 = x0 - i2 + C.yyyy;
  vec4 x3 = x0 - i3 + C.zzzz;
  vec4 x4 = x0 + C.wwww;

// Permutations
  i = mod289_4(i); 
  float j0 = permutef( permutef( permutef( permutef(i.w) + i.z) + i.y) + i.x);
  vec4 j1 = permute4( permute4( permute4( permute4 (
             i.w + (vec4){i1.w, i2.w, i3.w, 1.0f })
           + i.z + (vec4){i1.z, i2.z, i3.z, 1.0f })
           + i.y + (vec4){i1.y, i2.y, i3.y, 1.0f })
           + i.x + (vec4){i1.x, i2.x, i3.x, 1.0f });

// Gradients: 7x7x6 points over a cube, mapped onto a 4-cross polytope
// 7*7*6 = 294, which is close to the ring size 17*17 = 289.
  vec4 ip = (vec4){1.0f/294.0f, 1.0f/49.0f, 1.0f/7.0f, 0.0f} ;

  vec4 p0 = grad4(j0,   ip);
  vec4 p1 = grad4(j1.x, ip);
  vec4 p2 = grad4(j1.y, ip);
  vec4 p3 = grad4(j1.z, ip);
  vec4 p4 = grad4(j1.w, ip);

// Normalise gradients
  vec4 norm = taylorInvSqrt((vec4){dot(p0,p0), dot(p1,p1), dot(p2, p2), dot(p3,p3)});
  p0 *= norm.x;
  p1 *= norm.y;
  p2 *= norm.z;
  p3 *= norm.w;
  p4 *= taylorInvSqrtf(dot(p4,p4));

// Mix contributions from the five corners
  vec3 m0 = max(0.6f - (vec3){dot(x0,x0), dot(x1,x1), dot(x2,x2)}, 0.0);
  vec2 m1 = max(0.6f - (vec2){dot(x3,x3), dot(x4,x4)            }, 0.0);
  m0 = m0 * m0;
  m1 = m1 * m1;
  return 49.0f * ( dot(m0*m0, (vec3){ dot( p0, x0 ), dot( p1, x1 ), dot( p2, x2 )})
               + dot(m1*m1, (vec2){ dot( p3, x3 ), dot( p4, x4 ) } ) ) ;

}

