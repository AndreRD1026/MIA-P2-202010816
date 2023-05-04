import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';

@Injectable({
  providedIn: 'root'
})
export class ProyectoService {

  // constructor() { }
  constructor(private httpClient: HttpClient) { }

  // postEntrada(entrada: string) {
  //   return this.httpClient.post("http://localhost:5000/analizar", { Cmd: entrada });
  // }

  postEntrada(entrada: string) {
    return this.httpClient.post("http://44.211.150.86:5000/analizar", { Cmd: entrada });
  }

  // postEntrada(entrada: string) {
  //   return this.httpClient.post("http://192.168.1.15:5000/analizar", { Cmd: entrada });
  // }

  postLogin(id: string, user: string, pass: string) {
    const body = {
      id: id,
      user: user,
      pass: pass
    };
    return this.httpClient.post("http://44.211.150.86:5000/login", body);
  }

  // postLogin(id: string, user: string, pass: string) {
  //   const body = {
  //     id: id,
  //     user: user,
  //     pass: pass
  //   };
  //   return this.httpClient.post("http://192.168.1.15:5000/login", body);
  // }

  postLogout(){
    let id = ""
    return this.httpClient.post("http://44.211.150.86:5000/logout", id);
  }

  // postLogout(){
  //   let id = ""
  //   return this.httpClient.post("http://192.168.1.15:5000/logout", id);
  // }

  postRepDisk(id: string, ruta: string){
    const body = {
      id: id,
      ruta : ruta
    };
    return this.httpClient.post("http://44.211.150.86:5000/repDisk", body);
  }


  // postRepDisk(id: string, ruta: string){
  //   const body = {
  //     id: id,
  //     ruta : ruta
  //   };
  //   return this.httpClient.post("http://192.168.1.15:5000/repDisk", body);
  // }

  postRepSB(id: string, ruta: string){
    const body = {
      id: id,
      ruta : ruta
    };
    return this.httpClient.post("http://44.211.150.86:5000/repSB", body);
  }


  // postRepSB(id: string, ruta: string){
  //   const body = {
  //     id: id,
  //     ruta : ruta
  //   };
  //   return this.httpClient.post("http://192.168.1.15:5000/repSB", body);
  // }

  postRepTree(id: string, ruta: string){
    const body = {
      id: id,
      ruta : ruta
    };
    return this.httpClient.post("http://44.211.150.86:5000/repTree", body);
  }


  // postRepTree(id: string, ruta: string){
  //   const body = {
  //     id: id,
  //     ruta : ruta
  //   };
  //   return this.httpClient.post("http://192.168.1.15:5000/repTree", body);
  // }

  postRepFile(id: string, ruta: string){
    const body = {
      id: id,
      ruta : ruta
    };
    return this.httpClient.post("http://44.211.150.86:5000/repFile", body);
  }

  // postRepFile(id: string, ruta: string){
  //   const body = {
  //     id: id,
  //     ruta : ruta
  //   };
  //   return this.httpClient.post("http://192.168.1.15:5000/repFile", body);
  // }

}
