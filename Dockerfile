FROM mcr.microsoft.com/dotnet/sdk:8.0 AS build-env
WORKDIR /StickerManBot

# Copy everything
COPY . ./
# Restore as distinct layers
RUN dotnet restore
# Build and publish a release
RUN dotnet publish StickerManBot.csproj -c Release -o out

# Build runtime image
FROM mcr.microsoft.com/dotnet/aspnet:8.0
WORKDIR /StickerManBot
COPY --from=build-env /StickerManBot/out .
ENV DOTNET_EnableDiagnostics=0
EXPOSE 7592
ENTRYPOINT ["dotnet", "StickerManBot.dll"]
