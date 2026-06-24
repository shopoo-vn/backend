# Generate the RS256 keypair used by the Auth Service to sign JWTs.
# Other services only need keys\jwt_public.pem to verify tokens.
$ErrorActionPreference = "Stop"
$dir = Join-Path $PSScriptRoot "..\keys"
New-Item -ItemType Directory -Force -Path $dir | Out-Null
$priv = Join-Path $dir "jwt_private.pem"
$pub  = Join-Path $dir "jwt_public.pem"
openssl genpkey -algorithm RSA -out $priv -pkeyopt rsa_keygen_bits:2048
openssl rsa -in $priv -pubout -out $pub
Write-Host "RS256 keypair written to $dir"
