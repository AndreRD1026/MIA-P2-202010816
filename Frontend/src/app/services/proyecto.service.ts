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

  //18.212.40.106

  postEntrada(entrada: string) {
    return this.httpClient.post("http://44.201.152.11:5000/analizar", { Cmd: entrada });
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
    return this.httpClient.post("http://44.201.152.11:5000/login", body);
  }

  // postLogin(id: string, user: string, pass: string) {
  //   const body = {
  //     id: id,
  //     user: user,
  //     pass: pass
  //   };
  //   return this.httpClient.post("http://localhost:5000/login", body);
  // }

  postLogout(){
    let id = ""
    return this.httpClient.post("http://44.201.152.11:5000/logout", id);
  }

  // postLogout(){
  //   let id = ""
  //   return this.httpClient.post("http://localhost:5000/logout", id);
  // }

  // postLogout() {
  //   return this.httpClient.post("http://localhost:5000/logout")
  //     .pipe(
  //       map((response: any) => {
  //         return response.result;
  //       })
  //     );
  // }
}
